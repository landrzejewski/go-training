1. Zaimplementuj strukturę monetaryAmount, która opisuje wartości walutowe (kwota, waluta).
   Struktura powinna umożliwiać dodawanie i odejmowanie innych wartości walutowych (metody add, subtract).
   Zasygnalizuj błąd (zwróć wartość typu error) przy próbie wykonania operacji na różnych walutach.
   Uwaga: Go nie ma wyjątków - błędy są zwykłymi wartościami zwracanymi jako ostatni wynik funkcji,
   a panic/recover nie służy do obsługi spodziewanych sytuacji. Dodaj funkcję konstruktora.

2. W pakiecie common, zaimplementuj stos, przechowujący elementy typu int, oferujący operacje Push, Pop, Size.

3. Napisz grę w kółko i krzyżyk
   Plansza ma wymiary 3 x 3 pola.
   Gracze na zmianę zajmują wolne pola, umieszczając na nich swój znak (kółko lub krzyżyk).
   Gra kończy się, gdy wszystkie pola zostaną zajęte lub jeden z graczy zajmie zwycięską sekwencję (kolumnę, rząd lub przekątną).
   Interfejs gry powinien opierać się na wierszu poleceń/terminalu.

4. Napisz aplikację do rejestrowania wpływów/wydatków na potrzeby budżetu domowego.
   Aplikacja powinna rejestrować kwotę, rodzaj operacji, timestamp i jej opis (podane jako argumenty wiersza poleceń)
   oraz generować raport/tabelę w terminalu. Raport powinien zawierać wszystkie operacje i podsumowanie/saldo końcowe.
   Aplikacja powinna zapisywać dane wprowadzone przez użytkownika w pliku tekstowym (json).

5. Zaimplementuj poniższe narzędzia linii komend:
   - echo - drukuje tekst podany jako argumenty programu na standardowym wyjściu
   - cat - drukuje zawartość wskazanych plików na standardowym wyjściu,
     zezwala na opcjonalne numerowanie wierszy (przełącznik -n),
     numerowanie wierszy można wyłączyć dla pustych wierszy (przełącznik -nb)
   - find - przeszukuje i drukuje ścieżki plików i/lub katalogów, których nazwy pasują do wskazanego wzorca i typu,
     dozwolone typy to plik, katalog lub link symboliczny ("path/filepath" => filepath.WalkDir;
     filepath.Walk nadal działa, ale od Go 1.16 zalecany jest WalkDir - dostaje fs.DirEntry
     zamiast fs.FileInfo, więc unika wywołania stat dla każdego pliku)
   - grep - wyszukuje i drukuje wiersze zawierające wskazany tekst/wzorzec ze wskazanych plików/ścieżek

6. Bazując na pakiecie https://pkg.go.dev/net oraz wiedzy dotyczącej go routines/channels zbuduj czat pozwalający na:
   - komunikację jeden do wielu 
   - opcjonalnie:
      - rozmowy prywatne
      - zmianę statusu
      - przesyłanie plików

7. The Sleeping Barber dilemma, a classic computer science problem which illustrates the complexities that arise when
   there are multiple operating system processes. Here, we have a finite number of barbers, a finite number of seats in
   a waiting room, a fixed length of time the barbershop is open, and clients arriving at (roughly) regular intervals.
   When a barber has nothing to do, he or she checks the waiting room for new clients, and if one or more is there, a
   haircut takes place. Otherwise, the barber goes to sleep until a new client arrives. So the rules are as follows:
   - if there are no customers, the barber falls asleep in the chair
   - a customer must wake the barber if he is asleep
   - the customer leaves if all chairs are occupied and sits in an empty chair if it's available
   - when the barber finishes a haircut, he inspects the waiting room to see if there are any waiting customers
   and falls asleep if there are none
   - shop can stop accepting new clients at closing time, but the barbers cannot leave until the waiting room is empty
   - after the shop is closed and there are no clients left in the waiting area, the barber goes home
   The Sleeping Barber was originally proposed in 1965 by computer science pioneer Edsger Dijkstra.
   https://en.wikipedia.org/wiki/Sleeping_barber_problem

8. Stwórz bazę danych opartą o plik płaski przechowującą dane w postaci binarnej (https://gobyexample.com/reading-files).
   Baza powinna umożliwiać wykonywanie następujących operacji: ADD, READ, UPDATE, DELETE na podstawie podanego id rekordu.
   W celu uzyskania lepszej wydajności, wprowadź indeksowanie pozycji rekordu w pliku i/lub pamięć podręczną.
   Pomyśl o optymalnym sposobie usuwania rekordów i ponownym wykorzystaniem miejsca po usuniętym rekordzie.