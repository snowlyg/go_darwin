package router

import (
	"github.com/kataras/iris/v12"
	"github.com/rs/cors"
	"github.com/snowlyg/go_darwin/av"
	"github.com/snowlyg/go_darwin/models"
	"net"
)

type Server struct {
}

func (s *Server) Serve(l net.Listener) error {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))
	//app.Get("/", iris.FromStd(http.FileServer(FS)))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})
	// app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
	// 	[custom logic...]
	//  if shouldFireNetHTTPHandler {
	//    ...ServeHTTP(w,r)
	//    return
	//  }
	//  router(w,r)
	// })
	// In our case, the cors package has a ServeHTTP
	// of the same form of app.WrapRouter's accept input argument,
	// so we can just do:
	app.WrapRouter(c.ServeHTTP)

	// Register routes here...
	app.Get("/stage-api/article/list", listData)

	app.Run(iris.Listener(l))

	return nil
}

type TableList struct {
	Items []*models.Stream `json:"items"`
	Total int64            `json:"total"`
}

type PaginationParm struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func listData(ctx iris.Context) {

	var pp PaginationParm
	err := ctx.ReadQuery(&pp)
	if err != nil && !iris.IsErrPath(err) {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	streams, count := models.GetStreams(pp.Offset, pp.Limit)
	tableList := TableList{streams, count}

	ctx.JSON(tableList)
}

func NewServer(h av.Handler, rtmpAddr string) *Server {
	return &Server{}
}
