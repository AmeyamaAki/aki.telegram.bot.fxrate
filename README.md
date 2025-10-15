# aki.telegram.bot.fxrate

Another Telegram bot that provides foreign exchange rates.

In progress...

---

# Usage

Run `git clone https://github.com/AmeyamaAki/aki.telegram.bot.fxrate.git`, then `cd aki.telegram.bot.fxrate`.

Create docker-compose.yml in this directory.

```yaml
services:
  fxratebot:
    build: .
    environment:
      TELEGRAM_BOT_TOKEN: <your_bot_token>
    restart: unless-stopped
```

And then, use `docker compose up --build -d` to build and start the bot.

Send `/start` to the bot to get a list of commands.

You don't know bot token? Talk to [@BotFather](https://t.me/BotFather) on Telegram.

---

# Thanks

  - [186526/fxrate/](https://github.com/186526/fxrate/)
  - [go-telegram/bot](https://github.com/go-telegram/bot)
  - Bank of China
  - China Industrial Bank
  - Github Copilot