# Запуск бота «ЖКХ Контроль» MAX: пошаговая инструкция

Документ описывает первый рабочий контур:

```text
push в GitHub -> CI (тесты + линтер) -> ручной Deploy
             -> GHCR -> Docker Compose на VPS -> Caddy/HTTPS
             -> MAX отправляет сообщение -> бот отвечает тем же текстом
```

## 0. Что должно быть на руках

- GitHub-репозиторий проекта и права администратора репозитория.
- Домен, которым можно управлять через DNS.
- VPS с Ubuntu 22.04/24.04 и доступом root по SSH.
- Токен MAX-бота. Токен не публикуем в GitHub и не передаём в чат.
- Локально: Git и PowerShell. Docker Desktop нужен для локального запуска, но для первого деплоя достаточно Docker на VPS.

В примерах используются такие значения:

```text
bot.example.ru       # замените на свой поддомен
203.0.113.10         # замените на IP VPS
/opt/max-hackathon   # каталог приложения на VPS
deploy               # отдельный SSH-пользователь
```

## 1. Настроить DNS

В панели регистратора создайте запись:

| Тип | Имя | Значение |
|---|---|---|
| A | bot | публичный IPv4 VPS |

Итоговый адрес будет bot.example.ru. AAAA-запись добавляйте только если на VPS действительно настроен IPv6.

Проверьте с локального компьютера:

```powershell
nslookup bot.example.ru
```

## 2. Установить Docker на VPS

Подключитесь к серверу под root или пользователем с sudo:

```bash
ssh root@203.0.113.10
```

Выполните на Ubuntu:

```bash
apt update && apt upgrade -y
apt install -y ca-certificates curl ufw nano
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
cat >/etc/apt/sources.list.d/docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && printf '%s' "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
docker --version
docker compose version
docker run --rm hello-world
```

## 3. Создать отдельного пользователя для деплоя

На VPS:

```bash
adduser --disabled-password --gecos "" deploy
usermod -aG docker deploy
install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
```

На локальном Windows создайте отдельную пару ключей:

```powershell
$sshDir = Join-Path $env:USERPROFILE ".ssh"
New-Item -ItemType Directory -Force -Path $sshDir | Out-Null
ssh-keygen -t ed25519 -C "github-actions-max-hackathon" -f (Join-Path $sshDir "max_hackathon_deploy")
Get-Content (Join-Path $sshDir "max_hackathon_deploy.pub")
```

Скопируйте выведенную публичную строку на VPS:

```bash
nano /home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
```

Приватный файл max_hackathon_deploy не публикуйте и не добавляйте в Git.

Проверьте вход с Windows:

```powershell
ssh -i "$env:USERPROFILE\.ssh\max_hackathon_deploy" deploy@bot.example.ru
```

## 4. Открыть только нужные порты

На VPS:

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
ufw status
```

Порт 8080 наружу не открываем: backend доступен только внутри Docker-сети, а публичный HTTPS принимает Caddy.

## 5. Подготовить каталог и секрет MAX на VPS

Выполните на VPS:

```bash
mkdir -p /opt/max-hackathon/infra/caddy
chown -R deploy:deploy /opt/max-hackathon
```

Создайте файл /opt/max-hackathon/.env на VPS (команды выполняются под root):

```bash
install -m 600 -o deploy -g deploy /dev/null /opt/max-hackathon/.env
nano /opt/max-hackathon/.env
```

Содержимое (подставьте реальные значения):

```dotenv
MAX_BOT_TOKEN=сюда_токен_бота_MAX
PUBLIC_BASE_URL=https://bot.example.ru
MAX_WEBHOOK_SECRET=случайная_длинная_строка
```

Закройте доступ к файлу:

```bash
chmod 600 /opt/max-hackathon/.env
chown deploy:deploy /opt/max-hackathon/.env
```

Сейчас бот работает через Long Polling, поэтому MAX_WEBHOOK_SECRET не используется. Он оставлен для будущего перехода на Webhook. Реальный токен живёт только на VPS.

## 6. Заменить домен в Caddyfile

В файле infra/caddy/Caddyfile замените bot.example.com на настоящий домен:

```caddy
bot.example.ru {
    reverse_proxy bot:8080
}
```

Изменение нужно закоммитить и отправить в GitHub. Caddy сам запросит сертификат Let's Encrypt после того, как DNS уже указывает на VPS и порты 80/443 доступны.

## 7. Создать секреты GitHub Actions

В GitHub откройте Settings -> Environments -> New environment, создайте окружение production.

Добавьте в production -> Environment secrets:

| Secret | Значение |
|---|---|
| VPS_HOST | bot.example.ru или IP VPS |
| VPS_USER | deploy |
| VPS_APP_DIR | /opt/max-hackathon |
| VPS_SSH_PRIVATE_KEY | содержимое max_hackathon_deploy, включая BEGIN/END строки |
| VPS_KNOWN_HOSTS | вывод ssh-keyscan -H bot.example.ru |
| GHCR_USERNAME | GitHub username владельца токена |
| GHCR_READ_TOKEN | GitHub classic PAT с правом read:packages |

Получить known-hosts можно локально:

```powershell
ssh-keyscan -H bot.example.ru
```

Для публикации образа workflow использует встроенный GITHUB_TOKEN. GHCR_READ_TOKEN нужен VPS для скачивания приватного образа из GHCR. GitHub Packages сейчас требует Personal access token (classic), а не fine-grained token. Токен MAX через GitHub Actions не передаётся.

## 8. Проверить изменения и отправить в GitHub

Из корня проекта:

```powershell
git status
git add .
git commit -m "Prepare ЖКХ Контроль deployment"
git push origin main
```

Вкладка Actions должна показать workflow CI. Он проверяет форматирование, go vet, тесты с race detector, coverage, golangci-lint и сборку Docker-образа.

## 9. Выполнить первый деплой

1. Откройте GitHub -> Actions -> Deploy.
2. Нажмите Run workflow.
3. В поле ref укажите main (или вашу ветку).
4. Дождитесь зелёных job Build and publish image и Deploy to VPS.

Workflow собирает образ, публикует его в GHCR, копирует Compose/Caddyfile на VPS, запускает bot и Caddy, а затем удаляет старые образы.

## 10. Проверить HTTPS и контейнеры

Откройте:

```text
https://bot.example.ru/healthz
```

Ожидаемый ответ:

```text
ok
```

Для диагностики:

```bash
cd /opt/max-hackathon
docker compose --profile https ps
docker compose logs --tail=100 bot
docker compose logs --tail=100 caddy
```

## 11. Проверить бота в MAX

Откройте чат с ботом и отправьте:

```text
Привет, ЖКХ Контроль!
```

Бот должен ответить тем же текстом. Это Long Polling: HTTPS нужен для healthcheck и будущего развития, но не для доставки текущих сообщений.

## 12. Если не работает

- DNS/Caddy: nslookup bot.example.ru, затем docker compose logs caddy; проверьте A-запись и порты 80/443.
- Контейнер не стартует: проверьте /opt/max-hackathon/.env, права 600, имя MAX_BOT_TOKEN; затем docker compose logs bot.
- SSH из GitHub: проверьте VPS_USER, authorized_keys и VPS_KNOWN_HOSTS.
- Образ не скачивается: проверьте GHCR_USERNAME, PAT с read:packages и доступ к пакету.
- Бот молчит: проверьте токен, что запущен только один экземпляр бота, и ошибки MAX API в логах.

## 13. Что пока не нужно делать

- не открывать наружу порт 8080;
- не хранить токен в репозитории;
- не добавлять базу данных до появления подтверждённого сценария хранения;
- не писать frontend до решения о mini-app;
- для первого demo оставить Long Polling, но перед полноценным production-запуском перейти на Webhook.

После проверки можно развивать команды, роли, задания и фотоотчёты.

## Официальные ссылки

- [Подготовка и способы получения обновлений MAX](https://dev.max.ru/docs/chatbots/bots-coding/prepare)
- [Установка Docker Engine на Ubuntu](https://docs.docker.com/engine/install/ubuntu/)
- [GitHub Container Registry и права токенов](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Automatic HTTPS в Caddy](https://caddyserver.com/docs/automatic-https)
# Примечание об актуальности

Этот документ описывает первоначальный запуск бота и сохранён как историческая
инструкция. Актуальная архитектура приложения, Webhook, Tarantool, SeaweedFS,
роли и roadmap находятся в [PRODUCT_PASSPORT.md](PRODUCT_PASSPORT.md).
