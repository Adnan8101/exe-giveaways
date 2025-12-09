module.exports = {
  apps: [{
    name: "discord-giveaway-bot",
    script: "./discord-giveaway-bot",
    interpreter: "none", // Binary execution
    instances: 1,
    exec_mode: "fork",
    autorestart: true,
    watch: false,
    max_memory_restart: "1G",
    env: {
      NODE_ENV: "development",
    },
    env_production: {
      NODE_ENV: "production",
    },
    error_file: "./bot-error.log",
    out_file: "./bot.log",
    time: true // Add timestamp to logs
  }]
}
