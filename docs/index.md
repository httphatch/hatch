---
layout: home

hero:
  name: Hatch
  text: Local HTTPS development for macOS
  tagline: Automatic DNS, trusted certificates, and reverse proxying for your .test domains.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: What is Hatch?
      link: /guide/what-is-hatch

features:
  - title: Automatic DNS
    details: Resolves *.test domains to localhost automatically. No /etc/hosts editing required.
    icon: 🌐
  - title: Trusted HTTPS
    details: Generates a local CA trusted by macOS, so your browser shows a green lock with no warnings.
    icon: 🔒
  - title: Reverse Proxy
    details: Routes requests to your local services with path-based and subdomain-based routing.
    icon: 🔀
  - title: Dashboard
    details: Visual GUI for managing projects, viewing logs, and monitoring service health.
    icon: 📊
---

## Why Hatch?

If you're running more than one project locally, you've probably been there - trying to remember if your API is on port 3000 or 3001, which tab has the frontend, and why that other service won't start because something's already using 8080.

Hatch replaces all of that with clean `.test` domains. Instead of `localhost:3000`, you get `https://myapp.test`. Instead of guessing ports, every project gets its own domain with trusted HTTPS, automatic DNS, and proper routing, no matter how many things you're running at once.
