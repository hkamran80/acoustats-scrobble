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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/zmb3/spotify/v2"
)

type TrackDetails struct {
	URI      spotify.URI
	PlayedAt time.Time
}

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
	userId := os.Getenv("USER_ID")
	dbUri := os.Getenv("DB_URI")
	dbTableName := os.Getenv("DB_TABLE_NAME")

	return spotifyId != "" && spotifySecret != "" && fetchRange != "" && userId != "" && dbUri != "" && dbTableName != ""
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

	auth := spotifyauth.New(spotifyauth.WithRedirectURL("http://localhost:8080/callback"), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserReadRecentlyPlayed))
	state := strconv.FormatInt(time.Now().Unix(), 10)
	ctx := context.Background()

	client := Authenticate(ctx, *auth, state)

	user, err := client.CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Logged in as %s (%s)", user.ID, user.DisplayName)
	log.Printf("Loading recent tracks for this %s", fetchRange)

	var date time.Time
	currentDate := time.Now()
	if fetchRange == "year" {
		date = time.Date(currentDate.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	} else if fetchRange == "month" {
		date = time.Date(currentDate.Year(), currentDate.Month(), 1, 0, 0, 0, 0, time.Local)
	} else if fetchRange == "day" {
		date = time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.Local)
	}

	recentlyPlayedTracks, err := client.PlayerRecentlyPlayedOpt(ctx, &spotify.RecentlyPlayedOptions{Limit: 50, AfterEpochMs: date.Unix()})
	if err != nil {
		log.Fatal(err)
	}

	var trackHistory []TrackDetails

	for _, track := range recentlyPlayedTracks {
		trackHistory = append(trackHistory, TrackDetails{URI: track.Track.URI, PlayedAt: track.PlayedAt})
	}

	log.Printf("Loaded %d tracks from history!", len(trackHistory))

	conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URI"))
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close(context.Background())

	userId, err := uuid.Parse(os.Getenv("USER_ID"))
	if err != nil {
		log.Fatal(err)
	}

	

	copyCount, err := conn.CopyFrom(
		context.Background(),
		pgx.Identifier{os.Getenv("DB_TABLE_NAME")},
		[]string{"uri", "played_at", "user_id"},
		pgx.CopyFromSlice(len(trackHistory), func(i int) ([]any, error) {
			return []any{trackHistory[i].URI, trackHistory[i].PlayedAt.Format("2006-01-02T15:04:05-0700"), userId}, nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(copyCount)
}
