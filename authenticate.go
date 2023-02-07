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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"

	"github.com/zmb3/spotify/v2"
)

// TODO: Add environment variable option to change token path
const TokenFilePath = "token.json"

// Authenticate signs a user in
func Authenticate(ctx context.Context, auth spotifyauth.Authenticator, state string) *spotify.Client {
	var token *oauth2.Token

	loadedToken, tokenErr := loadToken()
	if tokenErr != nil {
		log.Println(tokenErr)

		newToken := getNewToken(ctx, auth, state)
		saveToken(newToken)

		log.Printf("Retrieved new token")
		token = newToken
	} else {
		log.Printf("Using cached token")
		token = loadedToken
	}

	client := spotify.New(auth.Client(ctx, token))
	return client
}

// getNewToken gets a new token from the Spotify API
func getNewToken(ctx context.Context, auth spotifyauth.Authenticator, state string) *oauth2.Token {
	authUrl := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", authUrl)
	fmt.Print("Please paste the URL you are redirected to here: ")

	reader := bufio.NewReader(os.Stdin)
	redirectedUrl, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	parsedRedirectedUrl, err := url.Parse(strings.TrimSpace(redirectedUrl))
	if err != nil {
		log.Fatal(err)
	}

	token, err := convertUrlToToken(ctx, *parsedRedirectedUrl, state, auth)
	if err != nil {
		log.Fatal(err)
	}

	return token
}

// A version of `spotifyauth`'s authentication implementation, except this one doesn't use `http.Request`
func convertUrlToToken(ctx context.Context, url url.URL, state string, auth spotifyauth.Authenticator) (*oauth2.Token, error) {
	values := url.Query()

	if e := values.Get("error"); e != "" {
		return nil, errors.New("spotify[custom]: auth failed - " + e)
	}

	code := values.Get("code")
	if code == "" {
		return nil, errors.New("spotify[custom]: didn't get access code")
	}

	actualState := values.Get("state")
	if actualState != state {
		return nil, errors.New("spotify[custom]: redirect state parameter doesn't match")
	}

	return auth.Exchange(ctx, code)
}

// saveToken saves an OAuth2 token to a JSON file named `token.json`
func saveToken(token *oauth2.Token) {
	file, jsonErr := json.MarshalIndent(token, "", " ")
	if jsonErr != nil {
		log.Println(jsonErr)
	}

	writeErr := os.WriteFile(TokenFilePath, file, 0644)
	if writeErr != nil {
		log.Println(writeErr)
	}
}

// loadToken loads an OAuth2 token from a JSON file named `token.json`
func loadToken() (*oauth2.Token, error) {
	file, readErr := os.ReadFile(TokenFilePath)
	if readErr != nil {
		return nil, readErr
	}

	token := &oauth2.Token{}
	jsonErr := json.Unmarshal(file, token)
	if jsonErr != nil {
		return nil, readErr
	}

	return token, nil
}
