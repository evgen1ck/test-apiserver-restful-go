package handlers_v1

import (
	"net/http"
	"test-server-go/internal/api_v1"
	"test-server-go/internal/storage"
	tl "test-server-go/internal/tools"
)

func (rs *Resolver) ProductsDataForMainpage(w http.ResponseWriter, r *http.Request) {
	// Block 1 - get products for mainpage
	products, err := storage.GetProductsForMainpage(r.Context(), rs.App.Postgres, rs.App.Config.App.Service.Url.Server)
	if err != nil {
		rs.App.Logger.NewWarn("error in get products for mainpage", err)
		api_v1.RespondWithInternalServerError(w)
		return
	}

	// Block 2 - send the result
	api_v1.RespondWithCreated(w, products)
}

func (rs *Resolver) ProductsData(w http.ResponseWriter, r *http.Request) {
	// Block 0 - get data
	textForSearch := r.URL.Query().Get("search")

	// Block 1 - get alternative search text variants
	transliterate := tl.Transliterate(textForSearch)
	rusToEng := tl.RusToEng(textForSearch)

	// Block 2 - get products with params
	products, err := storage.GetProductsWithParams(r.Context(), rs.App.Postgres, textForSearch, transliterate, rusToEng)
	if err != nil {
		rs.App.Logger.NewWarn("error in get products with params", err)
		api_v1.RespondWithInternalServerError(w)
		return
	}

	// Block 3 - send the result
	api_v1.RespondWithCreated(w, products)
}
