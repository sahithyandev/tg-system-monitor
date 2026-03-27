# tg-system-monitor

A simple and lightweight system monitor that interfaces through Telegram. Written in Go.

## Commands

### Build

```sh
go build -ldflags="-s -w" -o tgsm main.go
```

### Test

```sh
go test ./...
```

The above command will run tests all packages.

## Setup 

### Telegram
 
1. Create a Telegram bot using @BotFather.
2. Copy the bot token (looks like `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)
3. Replace `YOUR_BOT_TOKEN_HERE` with your actual bot token in the config file (located at `~/.config/tg-system-monitor/config.yml`).
4. Set the password for the bot using `set-password` command.

After that, the bot will connect and run on the bot.

A user can join the bot by using the `/join` command with the password.

## Author

Sahithyan K. (https://sahithyan.dev)