# Smart Theme

[![donation link](https://img.shields.io/badge/buy%20me%20a%20coffee-paypal-blue)](https://paypal.me/shaynejrtaylor?country.x=US&locale.x=en_US)

A simple theme that adjusts easily to a users preferences.

## Notice: This Theme Is Currently In Alpha

## Installation

```shell
git clone https://github.com/AspieSoft/smart-theme.git
cd smart-theme
go build
```

## Test Run

```shell
# listen on port 3000 and listen for file changes (with hot reload)
./compile 3000

# or listen for file changes without running an http server on localhost
./compile 0
```

## Config

In the src directory, you will find a config.yml file that you can modify.
Run ./compile to compile the config.yml file and to compile scripts and stylesheets for the theme, into the dist directory.
In the themes directory, you can add custom styles and scripts to add to the existing base theme.

## Setup

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8"/>
  <meta name="viewport" content="width=device-width, height=device-height, initial-scale=1.0, minimum-scale=1.0"/>

  <meta name="description" content="Your Description"/>
  <title>Your Title</title>

  <link rel="stylesheet" href="/theme/config.min.css"/>
  <link rel="stylesheet" href="/theme/style.norm.min.css"/>
  <script src="/theme/script.min.js" defer></script>

  <!--? optional: tailwind and htmx recommended -->
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
  <header>
    <div class="header-top">
      <a href="/" class="site-title">Your Logo</a>
      <!-- todo: add nav menu support here -->
    </div>
    <div class="header-image">
      <!--? optional element: nav menu -->
      <nav>
        <ul>
          <li><a href="/">Home</a></li>
          <li><a href="/about">About</a></li>
          <li><a href="/contact">Contact Us</a></li>
        </ul>
      </nav>
      <div class="content">
        <h1>Welcome To Your Title</h1>
      </div>
    </div>
  </header>
  <div id="page-content">
    <!--? optional class: page-bg -->
    <main class="content-grid page-bg">
      <h1>Hello, World!</h1>
      <p>Lorem ipsum dolor sit amet consectetur adipisicing elit. Aspernatur deleniti quidem alias delectus eius, ipsam molestiae? Dignissimos sequi eaque hic accusantium, molestias suscipit molestiae pariatur quisquam incidunt est nihil eveniet.</p>
      <p class="breakout">Lorem ipsum dolor sit amet consectetur adipisicing elit. Molestiae vero, cupiditate nulla esse nihil impedit voluptatum architecto autem incidunt animi? Neque, recusandae ipsum voluptatibus laudantium quod at ullam similique sequi.</p>
      <p>Lorem ipsum dolor sit amet consectetur adipisicing elit. Esse, exercitationem sunt nihil eum dolorum dolor culpa! Accusamus iusto ex nisi et doloremque expedita, voluptatem perferendis soluta iure omnis rerum sit.</p>
      <img src="/assets/background/falcon-desert.webp" alt="millennium falcon"/>
      <p>Lorem ipsum dolor sit amet consectetur adipisicing elit. Aspernatur deleniti quidem alias delectus eius, ipsam molestiae? Dignissimos sequi eaque hic accusantium, molestias suscipit molestiae pariatur quisquam incidunt est nihil eveniet.</p>
      <p class="breakout">Lorem ipsum dolor sit amet consectetur adipisicing elit. Molestiae vero, cupiditate nulla esse nihil impedit voluptatum architecto autem incidunt animi? Neque, recusandae ipsum voluptatibus laudantium quod at ullam similique sequi.</p>
      <p>Lorem ipsum dolor sit amet consectetur adipisicing elit. Esse, exercitationem sunt nihil eum dolorum dolor culpa! Accusamus iusto ex nisi et doloremque expedita, voluptatem perferendis soluta iure omnis rerum sit.</p>
    </main>
  </div>
  <footer>
    <!--? optional element: footer content -->
    <div class="content">
      <h1>Footer Content</h1>
      <p>Lorem ipsum dolor sit amet, consectetur adipisicing elit. Modi, libero animi. Doloremque minus quidem quasi vitae voluptates, porro officiis praesentium cupiditate omnis eligendi assumenda neque non? Impedit eligendi ab aut! Lorem ipsum dolor sit amet consectetur adipisicing elit. Corporis, sapiente animi enim sed tempora itaque eveniet excepturi nisi nobis similique ad voluptatum tempore minima sit fugiat reiciendis sunt quasi ab.</p>
    </div>
    <div class="copyright">
      <!--? you can replace this with your copyright text -->
      Powered by <a href="https://aspiesoft.com">AspieSoft</a>
    </div>
  </footer>
</body>
</html>
```
