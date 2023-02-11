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

	"os"
	"strconv"
	"time"

	"log"

	"github.com/georgysavva/scany/v2/pgxscan"
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

type DBRow struct {
	ID       int8      `db:"id"`
	URI      string    `db:"uri"`
	PlayedAt string    `db:"played_at"`
	UserId   uuid.UUID `db:"user_id"`
}

func Contains(s []*DBRow, track TrackDetails) bool {
	for _, v := range s {
		if v.URI == string(track.URI) && v.PlayedAt == track.PlayedAt.Format("2006-01-02T15:04:05-0700") {
			return true
		}
	}

	return false
}

func checkIfEnvVarsLoaded() bool {
	spotifyId := os.Getenv("SPOTIFY_ID")
	spotifySecret := os.Getenv("SPOTIFY_SECRET")
	userId := os.Getenv("USER_ID")
	dbUri := os.Getenv("DB_URI")
	dbTableName := os.Getenv("DB_TABLE_NAME")

	return spotifyId != "" && spotifySecret != "" && userId != "" && dbUri != "" && dbTableName != ""
}

func main() {
	envVarsLoaded := checkIfEnvVarsLoaded()
	if !envVarsLoaded {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// TODO: Update redirect URL
	auth := spotifyauth.New(spotifyauth.WithRedirectURL("http://localhost:8080/callback"), spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserReadRecentlyPlayed))
	state := strconv.FormatInt(time.Now().Unix(), 10)
	ctx := context.Background()

	client := Authenticate(ctx, *auth, state)

	user, err := client.CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Logged in as %s (%s)", user.ID, user.DisplayName)

	recentlyPlayedTracks, err := client.PlayerRecentlyPlayedOpt(ctx, &spotify.RecentlyPlayedOptions{Limit: 50})
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

	var dbTrackHistory []*DBRow
	rows, _ := conn.Query(context.Background(), `SELECT * FROM `+os.Getenv("DB_TABLE_NAME")+` WHERE user_id = ($1) ORDER BY played_at DESC LIMIT 50;`, userId)
	if err := pgxscan.ScanAll(&dbTrackHistory, rows); err != nil {
		log.Fatal(err)
	}

	var remainingTracks []TrackDetails
	for _, track := range trackHistory {
		if !Contains(dbTrackHistory, track) {
			remainingTracks = append(remainingTracks, track)
		}
	}

	copyCount, err := conn.CopyFrom(
		context.Background(),
		pgx.Identifier{os.Getenv("DB_TABLE_NAME")},
		[]string{"uri", "played_at", "user_id"},
		pgx.CopyFromSlice(len(remainingTracks), func(i int) ([]any, error) {
			return []any{remainingTracks[i].URI, remainingTracks[i].PlayedAt.Format("2006-01-02T15:04:05-0700"), userId}, nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Uploaded %d tracks to database!", copyCount)
}
