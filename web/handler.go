package web

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/net/context"

	"github.com/kyleconroy/deckbrew/api"
	"github.com/kyleconroy/deckbrew/brew"
	"github.com/kyleconroy/deckbrew/config"
	_ "github.com/lib/pq"

	"goji.io"
	"goji.io/pat"
)

const tmpl = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:site" content="@wizards_magic" />
    <meta name="twitter:title" content="{{.Card.Name}}" />
    <meta name="twitter:description" content="{{.Card.Text}}" />
    <meta name="twitter:image" content="{{.Edition.ImageUrl}}" />
  </head>
  <body>
    <h1>{{.Card.Name}}</h1>
    <img src="{{.Edition.ImageUrl}}" alt="{{.Card.Name}}" />
  </body>
</html>
`

type Web struct {
	r brew.Reader
	t *template.Template
}

type CardPage struct {
	Card    brew.Card
	Edition brew.Edition
}

func (web *Web) HandleCard(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(pat.Param(ctx, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	u, _ := url.Parse(fmt.Sprintf("?multiverseid=%d", id))
	s, err, _ := api.ParseSearch(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cards, err := web.r.GetCards(ctx, s)
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if len(cards) == 0 {
		http.Error(w, "No cards found", http.StatusNotFound)
		return
	}

	cp := CardPage{Card: cards[0]}
	for _, e := range cards[0].Editions {
		if e.MultiverseId == id {
			cp.Edition = e
		}
	}

	if err = web.t.Execute(w, cp); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
}

func New(cfg *config.Config, r brew.Reader) http.Handler {
	mux := goji.NewMux()

	app := Web{
		r: r,
		t: template.Must(template.New("card").Parse(tmpl)),
	}

	// Setup middleware
	mux.UseC(api.Recover)
	mux.UseC(api.Tracing)

	mux.HandleFuncC(pat.Get("/mtg/cards/:id"), app.HandleCard)
	mux.Handle(pat.New("/*"), http.FileServer(http.Dir("./web/static/")))

	return mux
}
