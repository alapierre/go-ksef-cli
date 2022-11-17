# go-ksef-cli
simple KSeF Command Line Interface

# Instalacja

Pobrać i rozlakować archiwum dla wybranego systemu operacyjnego (obecnie Linux x64 lub Windows x64) ze strony https://github.com/alapierre/go-ksef-cli/releases

# Konfiguracja

Plik `config.env` zawiera dostępne opcje konfiguracyjne. 

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