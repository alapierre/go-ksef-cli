# go-ksef-cli
Simple KSeF Command Line Interface

# Instalacja

- pobrać i rozpakować archiwum dla wybranego systemu operacyjnego (obecnie Linux x64 lub Windows x64) ze strony https://github.com/alapierre/go-ksef-cli/releases
- dostosować opcje konfiguracyjne, w szczególności ścieżka do kluczy, środowisko (test, demo, prod) 

# Konfiguracja

Plik `config.env` zawiera dostępne opcje konfiguracyjne. Plik może zostać zapisany w jednej s z dwóch lokalizacji:

1. w katalogu domowym użytkownika  `$HOME/.go-ksef-cli/config.env` - ta lokalizacja ma priorytet
2. w katalogu, z którego uruchamiana jest aplikacja `config.env` - w drugiej kolejności aplikacja szuka tutaj

# Przechowywanie tokena autoryzacyjnego

Aplikacja przechowuje token autoryzacyjny w postaci zaszyfrowanej w pliku zapisanym w katalogu domowym użytkownika. Klucz szyfrowania zapisany
jest w systemowym zasobniku haseł. Przed zapisaniem tokena, należy zainicjować klucz i go zapisać za pomocą polecenia:

```shell
ksef-cli init
```

Następnie można zapisać token:

```shell
ksef-cli store -t __token_autoryzacyjny___ -i __nip___
```

Tokeny dla różnych środowisk (test, demo, prod) są zapisywane w odrębnych katalogach w `$USER_HOME/.go-ksef-cli`

# Logowanie się do ksef

Jeśli token autoryzacyjny nie został zapisany

```shell
ksef-cli login -t __token_autoryzacyjny___ -i __nip___
```

Jeśli wcześniej zapisano token autoryzacyjny

```shell
ksef-cli login -i __nip___
```

# Zakończenie sesji

```shell
ksef-cli logout
```

Zakończenie innej sesji niż ostatnio otwarta

```shell
ksef-cli logout -t __token_sesjny__
```
