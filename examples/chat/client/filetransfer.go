package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"training.pl/go/examples/chat/common"
)

// ChunkSize is defined in common/constants.go as FileChunkSize

// FileTransfer manages file transfers.
//
// An outgoing transfer goes through two stages. SendFile registers a PENDING
// record under a local reference ID and sends the FILE init message; the server
// mints the real FileID and echoes the init back with both IDs, at which point
// startOutgoing re-keys the record and starts streaming chunks.
type FileTransfer struct {
	conn *Connection

	// pending holds outgoing transfers waiting for the server's echo, keyed by
	// the local reference ID. Guarded by conn.transfersMu like the main map.
	pending map[string]*pendingUpload
}

// pendingUpload is an outgoing transfer that has not received its FileID yet.
type pendingUpload struct {
	file        *os.File
	recipient   string
	totalChunks int
	progress    *FileTransferProgress
}

// NewFileTransfer creates a new file transfer manager
func NewFileTransfer(conn *Connection) *FileTransfer {
	return &FileTransfer{
		conn:    conn,
		pending: make(map[string]*pendingUpload),
	}
}

// SendFile sends a file to a recipient
func (ft *FileTransfer) SendFile(recipient, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	// NOTE: there is deliberately NO `defer file.Close()` here. SendFile returns
	// right after queuing the init message, so a defer would close the file before
	// sendFileChunks had read anything ("file already closed"). The descriptor is
	// owned by the pending record and later by the sendFileChunks goroutine; on
	// the error paths below it is closed explicitly.

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		file.Close()
		return fmt.Errorf("cannot send directory as file")
	}

	filename := filepath.Base(filePath)
	filesize := fileInfo.Size()

	// Validate file size. An empty file has no chunks to send, and the server
	// rejects a zero size anyway - say so up front rather than after a round trip.
	if filesize == 0 {
		file.Close()
		return fmt.Errorf("cannot send an empty file")
	}
	if filesize > common.MaxFileSize {
		file.Close()
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", common.MaxFileSize)
	}

	totalChunks := int(filesize / common.FileChunkSize)
	if filesize%common.FileChunkSize != 0 {
		totalChunks++
	}

	refID := generateRefID()
	upload := &pendingUpload{
		file:        file,
		recipient:   recipient,
		totalChunks: totalChunks,
		progress: &FileTransferProgress{
			FileID:      refID,
			Filename:    filename,
			Filesize:    filesize,
			IsIncoming:  false,
			StartTime:   time.Now(),
			totalChunks: totalChunks,
		},
	}

	ft.conn.transfersMu.Lock()
	ft.pending[refID] = upload
	ft.conn.transfersMu.Unlock()

	// Send file init message. The server assigns the FileID and echoes this
	// message back with RefID set - see startOutgoing.
	initMsg := &common.Message{
		Type:        common.TypeFile,
		Recipient:   recipient,
		RefID:       refID,
		Filename:    filename,
		Filesize:    filesize,
		TotalChunks: totalChunks,
		Timestamp:   time.Now(),
	}

	if err := ft.conn.enqueue(initMsg); err != nil {
		ft.abortPending(refID)
		return err
	}

	return nil
}

// startOutgoing is called when the server echoes our FILE init with the assigned
// FileID. It moves the pending record into the active transfer map under that ID
// and starts streaming the chunks.
func (ft *FileTransfer) startOutgoing(refID, fileID string) {
	ft.conn.transfersMu.Lock()
	upload, ok := ft.pending[refID]
	if !ok {
		ft.conn.transfersMu.Unlock()
		return
	}
	delete(ft.pending, refID)
	upload.progress.FileID = fileID
	ft.conn.fileTransfers[fileID] = upload.progress
	ft.conn.transfersMu.Unlock()

	go ft.sendFileChunks(upload.file, fileID, upload.recipient, upload.totalChunks)
}

// abortPending drops a pending upload (the server rejected it, or the init
// could not be queued) and closes its file.
func (ft *FileTransfer) abortPending(refID string) {
	ft.conn.transfersMu.Lock()
	upload, ok := ft.pending[refID]
	delete(ft.pending, refID)
	ft.conn.transfersMu.Unlock()
	if ok {
		upload.file.Close()
	}
}

// sendFileChunks sends file chunks. This goroutine is the sole owner of the file
// descriptor, so it is the one that closes it.
//
// There is no artificial delay between chunks: the bounded sendChan and TCP
// itself provide the flow control - enqueue blocks when writePump is behind.
func (ft *FileTransfer) sendFileChunks(file *os.File, fileID, recipient string, totalChunks int) {
	defer file.Close()

	buffer := make([]byte, common.FileChunkSize)
	chunkNum := 0

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			ft.notifyError(fileID, fmt.Sprintf("Read error: %v", err))
			return
		}

		if n == 0 {
			break
		}

		// Copy the chunk. The message is only QUEUED - writePump serialises it
		// later, in another goroutine, by which time the loop has overwritten
		// `buffer` with the next read. Sharing the buffer was a data race and
		// corrupted the file on the receiving side.
		chunk := make([]byte, n)
		copy(chunk, buffer[:n])

		// Send chunk
		chunkMsg := &common.Message{
			Type:        common.TypeFileChunk,
			Recipient:   recipient,
			FileID:      fileID,
			ChunkNum:    chunkNum,
			TotalChunks: totalChunks,
			Data:        chunk,
			Timestamp:   time.Now(),
		}

		if err := ft.conn.enqueue(chunkMsg); err != nil {
			ft.notifyError(fileID, err.Error())
			return
		}

		// Update progress
		ft.updateProgress(fileID, chunkNum, totalChunks)

		chunkNum++
	}

	// File transfer complete
	ft.notifyComplete(fileID)
}

// IsIncoming reports whether a transfer is an INCOMING one. The server sends
// FileComplete to both sides, but only the receiver has anything to write.
func (ft *FileTransfer) IsIncoming(fileID string) bool {
	transfer, exists := ft.conn.transfer(fileID)
	return exists && transfer.IsIncoming
}

// ReceiveFile saves a received file and returns the path it was written to.
func (ft *FileTransfer) ReceiveFile(fileID string) (string, error) {
	transfer, exists := ft.conn.transfer(fileID)
	if !exists {
		return "", errors.New("file transfer not found")
	}

	// Create downloads directory
	downloadDir := filepath.Join(".", "downloads")
	if err := os.MkdirAll(downloadDir, common.DirMode()); err != nil {
		return "", fmt.Errorf("failed to create download directory: %v", err)
	}

	// Sanitize filename to prevent path traversal attacks
	filename := filepath.Base(transfer.Filename)
	if filename == "." || filename == ".." || filename == "/" || filename == "" {
		return "", fmt.Errorf("invalid filename: %s", transfer.Filename)
	}

	// Snapshot the chunks under the record's lock. The readPump goroutine
	// (Connection.handleFileChunk) writes to the same map while ReceiveFile is
	// called from the UI goroutine - reading and writing a map at once is a race,
	// and in the worst case "fatal error: concurrent map read and map write".
	chunks, missing := transfer.orderedChunks()
	if missing >= 0 {
		return "", fmt.Errorf("missing chunk %d", missing)
	}

	// Create the file without clobbering an existing one of the same name.
	file, filePath, err := createUnique(downloadDir, filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write chunks in order
	for _, chunk := range chunks {
		if _, err := file.Write(chunk); err != nil {
			return "", fmt.Errorf("failed to write chunk: %v", err)
		}
	}

	// Clean up
	ft.conn.removeTransfer(fileID)

	return filePath, nil
}

// createUnique creates dir/name, or dir/name (1), dir/name (2), ... when the
// plain name already exists, so a second download never overwrites the first.
func createUnique(dir, name string) (*os.File, string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 0; ; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		path := filepath.Join(dir, candidate)
		// O_EXCL makes the existence check and the create one atomic operation.
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, common.FileMode())
		if err == nil {
			return f, path, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}
}

// updateProgress updates transfer progress
func (ft *FileTransfer) updateProgress(fileID string, chunkNum, totalChunks int) {
	if transfer, exists := ft.conn.transfer(fileID); exists {
		transfer.setProgress(float64(chunkNum+1) / float64(totalChunks) * 100)
	}
}

// notifyComplete notifies completion
func (ft *FileTransfer) notifyComplete(fileID string) {
	if transfer, exists := ft.conn.removeTransfer(fileID); exists {
		duration := time.Since(transfer.StartTime)
		speed := float64(transfer.Filesize) / duration.Seconds() / 1024 / 1024 // MB/s

		fmt.Printf("\nFile transfer complete: %s (%.2f MB/s)\n", transfer.Filename, speed)
	}
}

// notifyError notifies transfer error
func (ft *FileTransfer) notifyError(fileID, reason string) {
	if transfer, exists := ft.conn.removeTransfer(fileID); exists {
		fmt.Printf("\nFile transfer error: %s - %s\n", transfer.Filename, reason)
	}
}

// TransferProgress returns current transfer progress
func (ft *FileTransfer) TransferProgress() []string {
	ft.conn.transfersMu.Lock()
	transfers := make([]*FileTransferProgress, 0, len(ft.conn.fileTransfers)+len(ft.pending))
	for _, transfer := range ft.conn.fileTransfers {
		transfers = append(transfers, transfer)
	}
	for _, upload := range ft.pending {
		transfers = append(transfers, upload.progress)
	}
	ft.conn.transfersMu.Unlock()

	var progress []string
	for _, transfer := range transfers {
		direction := "↓"
		if !transfer.IsIncoming {
			direction = "↑"
		}

		status := fmt.Sprintf("%s %s: %.1f%% (%s)",
			direction,
			transfer.Filename,
			transfer.Progress(),
			formatFileSize(transfer.Filesize))

		progress = append(progress, status)
	}

	return progress
}

// generateRefID generates a local reference for a pending upload
func generateRefID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
