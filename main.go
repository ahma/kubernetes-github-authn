package main

import (
	"context"
	"encoding/json"
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
			log.Println("[Error]", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apiVersion": "authentication.k8s.io/v1beta1",
				"kind":       "TokenReview",
				"status": authentication.TokenReviewStatus{
					Authenticated: false,
				},
			})
			return
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tr.Spec.Token},
		)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		client := github.NewClient(tc)

		GithEntUrl := os.Getenv("GITHUB_ENTERPRISE_URL")
		if len(GithEntUrl) != 0 {
			baseURL, err := url.ParseRequestURI(GithEntUrl)
			if err != nil {
				log.Fatalln("Base URL ERROR:", err)
			}
			client.BaseURL = baseURL
		}

		user, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			log.Println("[Error]", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apiVersion": "authentication.k8s.io/v1beta1",
				"kind":       "TokenReview",
				"status": authentication.TokenReviewStatus{
					Authenticated: false,
				},
			})
			return
		}

		GithOrgs := os.Getenv("GITHUB_ORGANISATIONS")
		if len(GithOrgs) != 0 {
			result := strings.Split(GithOrgs, ",")
			for i := range result {
				org, _, _ := client.Organizations.GetOrgMembership(context.Background(), *user.Login, result[i])

				if org != nil {
					os.Setenv("GITHUB_AUTH_TYPE", "mail")

					K8sUser := *user.Login
					GithAuthType := os.Getenv("GITHUB_AUTH_TYPE")
					if len(GithAuthType) != 0 {
						if GithAuthType == "mail" {
							K8sUser = *user.Email
						}
					}

					log.Printf("[Success] login as %s", K8sUser)
					w.WriteHeader(http.StatusOK)

					trs := authentication.TokenReviewStatus{
						Authenticated: true,
						User: authentication.UserInfo{
							Username: K8sUser,
							UID:      *user.Login,
						},
					}
					json.NewEncoder(w).Encode(map[string]interface{}{
						"apiVersion": "authentication.k8s.io/v1beta1",
						"kind":       "TokenReview",
						"status":     trs,
					})

					return
				}
			}
			log.Println("[Error] User not in Organisations!")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"apiVersion": "authentication.k8s.io/v1beta1",
				"kind":       "TokenReview",
				"status": authentication.TokenReviewStatus{
					Authenticated: false,
				},
			})
			return

		}

	})
	log.Fatal(http.ListenAndServe(":3000", nil))
}
