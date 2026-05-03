# URL Shortener (Week 1)

## Overview

A simple URL shortener built using Go and containerized using Docker.

This version uses an in-memory key-value store to store shortened URLs.

---

## Features

- HTML form for URL input
- SHA256-based short code generation (first 8 characters)
- Redirect support
- Input validation
- Dockerized application

---

## Tech Stack

- Go 1.24
- Docker
- Standard Library (net/http, html/template, crypto/sha256)

---

## How to Run Locally

### Without Docker

```bash
go run main.go