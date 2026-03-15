# INSTALLATION

In a go project run:
```
go get github.com/TheAntiFish/Chirpy
```

# REQUIREMENTS
A .env file at the root of the module containing the following variables:
- DB_URL: The URL to where the database is stored
- PLATFORM: The current platform
- SECRET: A token secret used to create and validate JWT's
- POLKA_KEY: An API key to access Polka
- DIR: The filepath where your HTML files are stored

# FEATURES
- /app/: The prefix for accessing the HTML file directory
- GET /api/healthz: Returns OK

Users:
- POST /api/users: Creates a user in the database with given Username and Password
- POST /api/login: Logs in user with given Username and Password
- PUT /api/users: Update currently logged in user with given Username and Password

Chirps:
- POST /api/chirps: Creates a chirp for the current user, given the chirp body
- GET /api/chirps: Gets all chirps sorted by post date, optional parameters to limit chirps to a specific user or sort chirps desc
- Get /api/chirps/{id}: Gets a specific chirp
- DELETE /api/chirps/{id}: Deletes a chirp

Tokens:
- POST /api/refresh: Returns an auth token if the user has a valid refresh token
- POST /api/revoke: Revokes a users refresh token

Webhooks:
- POST /api/polka/webhooks: Used by polka to set users ChirpRed status

Admin:
- GET /admin/metrics: Get the amount of times the webpage has been visited.
- POST /admin/reset: Reset the amount of times the webpage has been visited and clear the database, only usable if .env variable PLATFORM is "dev"