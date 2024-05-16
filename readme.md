# Go Web Server Template

A Template for a basic webserver in golang.

## Installation

```shell script
git clone https://github.com/AspieSoft/go-webserver
```

## Files

- app.go: default file for gofiber page rendering.
- db.go: initialize the database.
- login.go: everything related to the login and verification.
- main.go: starts the server.
- website/config.yml: a configuration file for handling the domain and ssl.
- website/db: database files and other dynamically generated files that should Not be touched manually.
- website/html: handlebars page templates.
- website/assets: static assets.
- website/theme: a clone of [smart-theme](https://github.com/AspieSoft/smart-theme) for more advanced themeing.
- website/theme/src/config.yml: theme configuration.
