1. Zaimplementuj strukturę monetaryAmount, która opisuje wartości walutowe (kwota, waluta).
   Struktura powinna umożliwiać dodawanie i odejmowanie innych wartości walutowych (metody add, subtract).
   Obsłuż wyjątek wynikający z próby wykonania operacji na różnych walutach. Dodaj funkcję konstruktora.

2. Napisz grę w kółko i krzyżyk
   Plansza ma wymiary 3 x 3 pola.
   Gracze na zmianę zajmują wolne pola, umieszczając na nich swój znak (kółko lub krzyżyk).
   Gra kończy się, gdy wszystkie pola zostaną zajęte lub jeden z graczy zajmie zwycięską sekwencję (kolumnę, rząd lub przekątną).
   Interfejs gry powinien opierać się na wierszu poleceń/terminalu.

3. W pakiecie common, zaimplementuj stos, przechowujący elementy typu int, oferujący operacje Push, Pop, Size.

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
     dozwolone typy to plik, katalog lub link symboliczny ("path/filepath" => filepath.Walk)
   - grep - wyszukuje i drukuje wiersze zawierające wskazany tekst/wzorzec ze wskazanych plików/ścieżek