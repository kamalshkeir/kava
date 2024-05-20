package kava

import (
	"net/http"
	"strings"
)

func Handler(gen *Generator, genOpts GenerateOpts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		initials := r.URL.Query().Get(gen.queryParamName)
		if initials == "" {
			initials = r.Context().Value(gen.queryParamName).(string)
		}
		// Generate the image
		if len(initials) < 4 {
			genOpts.Text = strings.ToUpper(initials)
		} else {
			sp := strings.Split(initials, "-")
			letters := ""
			for _, v := range sp {
				if v != "" {
					letters += strings.ToUpper(string(v[0]))
				}
			}
			if len(letters) > 3 {
				letters = letters[:3]
			}
			if letters != "" {
				genOpts.Text = letters
			} else {
				genOpts.Text = "--"
			}
		}

		if err := gen.Generate(genOpts); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
