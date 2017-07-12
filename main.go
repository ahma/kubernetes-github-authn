package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	authentication "k8s.io/client-go/pkg/apis/authentication/v1beta1"
)

func main() {
	http.HandleFunc("/authenticate", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var tr authentication.TokenReview
		err := decoder.Decode(&tr)
		if err != nil {
			AuthFailed(w, err)
			return
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tr.Spec.Token},
		)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		client := github.NewClient(tc)

		// os.Setenv("GITHUB_ENTERPRISE_URL", "www.google.com")
		GithEntUrl := os.Getenv("GITHUB_ENTERPRISE_URL")
		if len(GithEntUrl) != 0 {
			baseURL, err := url.ParseRequestURI(GithEntUrl)
			if err != nil {
				AuthFailed(w, err)
				return
			}
			client.BaseURL = baseURL
		}

		user, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			AuthFailed(w, err)
			return
		}

		var GithubUID string
		GithubUID = *user.Login

		var K8sUser string
		K8sUser = *user.Login

		// os.Setenv("GITHUB_AUTH_TYPE", "mail")
		GithAuthType := os.Getenv("GITHUB_AUTH_TYPE")
		if len(GithAuthType) != 0 {
			if GithAuthType == "mail" {
				K8sUser = *user.Email
			}
		}

		// os.Setenv("GITHUB_ORGANISATIONS", "dyninc")
		GithOrgs := os.Getenv("GITHUB_ORGANISATIONS")
		if len(GithOrgs) != 0 {
			result := strings.Split(GithOrgs, ",")
			for i := range result {
				org, _, err := client.Organizations.GetOrgMembership(context.Background(), *user.Login, result[i])
				if err != nil {

				}
				if org != nil {
					AuthCompleted(w, GithubUID, K8sUser)
					return

				}
			}
			err = errors.New("user not in organisations")
			AuthFailed(w, err)

		} else {
			AuthCompleted(w, GithubUID, K8sUser)
			return
		}

	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func AuthFailed(w http.ResponseWriter, err error) {
	log.Println("[Error]", err.Error())
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apiVersion": "authentication.k8s.io/v1beta1",
		"kind":       "TokenReview",
		"status": authentication.TokenReviewStatus{
			Authenticated: false,
		},
	})

}

func AuthCompleted(w http.ResponseWriter, GithubUID string, K8sUser string) {
	log.Printf("[Success] login as %s", K8sUser)
	w.WriteHeader(http.StatusOK)

	trs := authentication.TokenReviewStatus{
		Authenticated: true,
		User: authentication.UserInfo{
			Username: K8sUser,
			UID:      GithubUID,
		},
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apiVersion": "authentication.k8s.io/v1beta1",
		"kind":       "TokenReview",
		"status":     trs,
	})

}
