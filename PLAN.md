# MiniRedis Plan

- Hotove:
  - TCP server na porte z `MINIREDIS_PORT`.
  - In-memory key/value store so `sync.RWMutex`.
  - Prikazy `PING`, `STATUS`, `SET`, `GET`, `DELETE`, `QUIT`, `EXIT`.
  - Stress test pre vela klientov v `test/stress-test.go`.
  - Config balicek `internal/config` cita `.env`.
  - Docker Compose posiela config do kontajnera.
  - Docker Compose vie nastavit port, RAM limit a `nofile`.
  - Server pouziva `MINIREDIS_MAX_CLIENTS`.
  - Server pouziva `MINIREDIS_MAX_VALUE_BYTES`.
  - Server pouziva read/write timeouty.
  - Server pouziva global/IP rate limit.
  - Go runtime dostava memory target z `MINIREDIS_MEMORY_MB`.
  - Kod ma vysvetlujuce komentare v config, server, handler a protocol castiach.

- Graceful shutdown:
  - Zachytit `SIGINT` a `SIGTERM` v `cmd/server/main.go`.
  - Pridat `Server.Shutdown()` metodu.
  - Pri shutdown zavriet listener, aby `Accept()` skoncil.
  - Prestat prijimat novych klientov.
  - Existujucim klientom dat kratky cas na dokoncnie prikazu.
  - Po timeout-e zavriet aktivne spojenia.
  - Logovat `event=shutdown_start` a `event=shutdown_done`.
  - Otestovat, ze `Ctrl+C` ukonci server bez zaseknutia.

- HELP command:
  - Pridat `HELP` do parsera bez argumentov.
  - V handleri vratit kratky zoznam prikazov.
  - Udrzat odpoved jednoducho citatelnu cez `nc`.
  - Zahrnut prikazy: `PING`, `STATUS`, `SET`, `GET`, `DELETE`, `QUIT`, `EXIT`.
  - Po pridani novych prikazov aktualizovat aj `HELP`.
  - Doplnit test, ze `HELP` vrati text a nezavrie spojenie.

- Zjednotenie odpovedi protokolu:
  - Rozhodnut jednotny styl pre missing key.
  - Navrh: `GET missing` vrati `$NULL`.
  - Navrh: `DELETE missing` vrati `:0`.
  - Navrh: `DELETE existing` vrati `:1`.
  - Nechat chyby iba pre zly prikaz, zle argumenty, limit a server problem.
  - Aktualizovat README priklady po zmene.
  - Doplnit parser/handler testy pre vsetky odpovede.

- Redis-like prikazy:
  - `EXISTS key`: vrati `:1` ked kluc existuje, inak `:0`.
  - `DEL key`: alias pre `DELETE key`, odporucany Redis nazov.
  - `EXPIRE key seconds`: nastavi expiraciu existujuceho kluca.
  - `TTL key`: vrati zostavajuce sekundy, `-1` bez expiracie, `-2` ked kluc neexistuje.
  - `SETEX key seconds value`: ulozi hodnotu rovno s expiraciou.
  - Upravit store tak, aby vedel drzat hodnotu aj expire timestamp.
  - Pri `GET`, `EXISTS`, `DELETE`, `TTL` najprv odstranit expirovany kluc.
  - Neskor pridat background cleanup expirovanych klucov.
  - Doplnit testy pre expiracie a edge cases.

- README aktualizacia:
  - Popisat spustenie cez `go run ./cmd/server`.
  - Popisat spustenie cez `docker compose up --build`.
  - Vypisat `.env` premenne a co realne robia.
  - Vysvetlit rozdiel medzi `MINIREDIS_MEMORY_MB` a Docker `mem_limit`.
  - Pridat priklady cez `nc localhost 6379`.
  - Pridat zoznam podporovanych prikazov a odpovedi.
  - Pridat priklad stress testu.
  - Pridat poznamku, ze protokol este nie je plny Redis RESP.

- Testy:
  - Unit testy pre `internal/config`.
  - Unit testy pre `internal/store`.
  - Unit testy pre `internal/network/protocol.go`.
  - Integration test pre `HELP`.
  - Integration test pre max clients.
  - Integration test pre max value.
  - Integration test pre rate limit.
  - Integration test pre `GET missing` a `DELETE missing`.
  - Testy pre `EXISTS`, `DEL`, `TTL`, `EXPIRE`, `SETEX`.
