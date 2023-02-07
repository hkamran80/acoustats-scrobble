/*
	Acoustats - The easiest way to view statistics regarding your music streaming habits
    Copyright (C) 2023 H. Kamran (https://hkamran.com)

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published
    by the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"strings"

	"os"
	"strconv"
	"time"

	"log"

	"github.com/joho/godotenv"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	// "github.com/schollz/progressbar/v3"

	"github.com/zmb3/spotify/v2"
)

func Contains(s []spotify.ID, str spotify.ID) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func checkIfEnvVarsLoaded() bool {
	spotifyId := os.Getenv("SPOTIFY_ID")
	spotifySecret := os.Getenv("SPOTIFY_SECRET")
	fetchRange := os.Getenv("RANGE")

	return spotifyId != "" && spotifySecret != "" && fetchRange != ""
}

func main() {
	envVarsLoaded := checkIfEnvVarsLoaded()
	if !envVarsLoaded {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	fetchRange := strings.ToLower(os.Getenv("RANGE"))
	if fetchRange != "day" && fetchRange != "month" && fetchRange != "year" {
		log.Println("Invalid `fetchRange` parameter, defaulting to \"month\"...")
		fetchRange = "month"
	}

	auth := spotifyauth.New(spotifyauth.WithRedirectURL("http://localhost:8080/callback"), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopePlaylistReadCollaborative, spotifyauth.ScopePlaylistModifyPublic, spotifyauth.ScopePlaylistModifyPrivate))
	state := strconv.FormatInt(time.Now().Unix(), 10)
	ctx := context.Background()

	client := Authenticate(ctx, *auth, state)

	user, err := client.CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Logged in as %s (%s)", user.ID, user.DisplayName)
}
