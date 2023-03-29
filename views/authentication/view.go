// Package authentication contains the authentication-related views.
package authentication

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/nmcapule/oclz-go/oauth2"
	"github.com/nmcapule/oclz-go/syncer"
	"github.com/nmcapule/oclz-go/utils"
	"github.com/pocketbase/pocketbase"
)

//go:embed *.html
var fs embed.FS

// View is the main view for the authentication module.
type View struct {
	App         *pocketbase.PocketBase
	Syncer      *syncer.Syncer
	GroupPrefix string
}

func (v *View) Hook(parent *echo.Group) error {
	templates := template.Must(template.ParseFS(fs, "*.html"))

	base := parent.Group(v.GroupPrefix)
	base.GET("", func(c echo.Context) error {
		var buf bytes.Buffer
		if err := templates.ExecuteTemplate(&buf, "index.html", map[string]any{
			"Tenants": v.Syncer.Tenants,
		}); err != nil {
			return fmt.Errorf("executing template: %w", err)
		}
		return c.HTML(http.StatusOK, buf.String())
	})
	base.GET("/reauth/:name", func(c echo.Context) error {
		tenant := v.Syncer.Tenants[c.PathParam("name")]
		redirect := tenant.CredentialsManager().GenerateAuthorizationURL()
		return c.Redirect(http.StatusFound, redirect)
	})
	base.GET("/refresh/:name", func(c echo.Context) error {
		tenant := v.Syncer.Tenants[c.PathParam("name")]

		data := make(map[string]string)
		queries := c.QueryParams()
		for key := range queries {
			data[key] = queries.Get(key)
		}
		credentials, err := tenant.CredentialsManager().GenerateCredentials(*utils.GJSONFrom(data))
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("generating credentials: %v", err))
		}
		if credentials.RefreshToken == "" || credentials.AccessToken == "" {
			return c.String(
				http.StatusInternalServerError,
				fmt.Sprintf("got empty tokens! refresh_token=%q, access_token=%q",
					credentials.RefreshToken,
					credentials.AccessToken))
		}
		oauth2Service := oauth2.Service{Dao: v.Syncer.Dao}
		if err := oauth2Service.Save(credentials); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("saving credentials: %v", err))
		}

		return c.Redirect(http.StatusFound, "/authentication")
	})

	return nil
}
