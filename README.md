# Gondola быстро и в терминале

#### Для билда гондолы надо выполнить это:
```bash
mkdir -p bin && go build -o bin/gondola .
```

#### Далее заходим в примеры: *examples/sum-service* и в папке сервиса далаем это:
```bash
~/path/to/your/gondola/bin/gondola build # делаем билд сервиса
~/path/to/your/gondola/bin/gondola test # запускаем тесты на сервис
~/path/to/your/gondola/bin/gondola deploy # делаем деплой по ssh как systemd unit
```

#### Пример конфига и описание:
``` yaml
project:
  name: "myapp"

build:
  entry: "." # билд в текущей директории с файлом main.go
  output: "bin/myapp" # куда будет билдиться
  goos: "linux" # для какой системы: win/linux
  goarch: "amd64" # архитектура
  ldflags: "-s -w" # доп флаги, пока не понял зачем
  extra_env:
    - "CGO_ENABLED=0" # пока непонятно зачем это

test:
  commands:
    - "go test ./..." # запуск тестов
  skip: false

deploy:
  host: "your.host.com" # хост вм
  port: 22 # дефолтный 22 для ssh
  user: "youruser" # юзер вм
  key_path: "~/.ssh/your-ssh-private-key" # приватный ключ
  remote_path: "/opt/myapp/myapp" # путь на вм куда будем копировать бинарник
  service:
    name: "myapp" # название systemd unit
    description: "MyApp Service" # описание
    restart: "on-failure" # перезапуск при fail
  pre_deploy: [] # тут оставляем пустым если делаем деплой 1 раз, если больше чем 1 тогда надо сделать, чтобы он останавливал сервис
  post_deploy:
    - "chmod +x /opt/myapp/myapp" # даем права на myapp по полной
    - "sudo systemctl start myapp" # запускаем сервис
```