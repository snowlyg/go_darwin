package router

import (
	"github.com/kataras/iris/v12"
	"github.com/rs/cors"
	//"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/snowlyg/go_darwin/av"
	"github.com/snowlyg/go_darwin/client"
	"github.com/snowlyg/go_darwin/models"
	"net"
)

type Server struct {
}

func NewServer(h av.Handler, rtmpAddr string) *Server {
	return &Server{}
}

func (s *Server) Serve(l net.Listener) error {
	app := iris.Default()
	app.HandleDir("/static", iris.Dir("./www/dist/static"))
	app.RegisterView(iris.HTML("./www/dist", ".html"))

	app.Get("/", func(ctx iris.Context) {
		ctx.View("index.html")
	})

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		Debug:            false,
	})
	app.WrapRouter(c.ServeHTTP)
	app.Get("/stage-api/vue-element-admin/article/list", listData)
	app.Get("/stage-api/vue-element-admin/user/info", userInfo)
	app.Post("/stage-api/vue-element-admin/user/login", login)
	app.Post("/stage-api/vue-element-admin/article/create", create)
	app.Delete("/stage-api/vue-element-admin/article/{id:uint}", delete)
	app.Get("/stage-api/vue-element-admin/article/start/{id:uint}", start)
	app.Get("/stage-api/vue-element-admin/article/stop/{id:uint}", stop)

	err := app.Run(iris.Listener(l))
	if err != nil {
		return err
	}

	return nil
}

type Req struct {
	Data interface{} `json:"data"`
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
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
	req := Req{tableList, 20000, ""}
	ctx.JSON(req)
}

type LoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenReq struct {
	Token string `json:"token"`
}

func login(ctx iris.Context) {
	var loginInfo LoginInfo
	err := ctx.ReadForm(&loginInfo)
	if err != nil {
		if !iris.IsErrPath(err) /* see: https://github.com/kataras/iris/issues/1157 */ ||
			err == iris.ErrEmptyForm {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
	}

	reqToken := TokenReq{"admin-token"}
	req := Req{reqToken, 20000, ""}
	ctx.JSON(req)
}

type Admin struct {
	Roles        []string `json:"roles"`
	Avatar       string   `json:"avatar"`       //"https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif"
	Introduction string   `json:"introduction"` // "I am a super administrator"
	Name         string   `json:"name"`         // "Super Admin"
}

func userInfo(ctx iris.Context) {
	admin := Admin{[]string{"admin"}, "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif", "I am a super administrator", "Super Admin"}
	req := Req{admin, 20000, ""}
	ctx.JSON(req)
}

type StreamCreate struct {
	RoomName string `json:"roomName"`
	Source   string `json:"source"`
}

func create(ctx iris.Context) {
	req := Req{nil, 20000, "添加成功"}
	var streamCreate StreamCreate
	err := ctx.ReadJSON(&streamCreate)
	if err != nil {
		if !iris.IsErrPath(err) || err == iris.ErrEmptyForm {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
	}

	stream, err := models.AddStream(streamCreate.Source, streamCreate.RoomName)
	if err != nil {
		req.Msg = err.Error()
		ctx.JSON(req)
		return
	}
	if stream.ID == 0 {
		req.Msg = "添加失败"
		ctx.JSON(req)

		return
	}

	ctx.JSON(req)

}

func start(ctx iris.Context) {
	req := Req{nil, 20000, "启动拉流"}
	id := ctx.Params().GetUintDefault("id", 0)
	stream, err := models.StartStream(id)
	if err != nil {
		req.Msg = err.Error()
		ctx.JSON(req)
		return
	}

	if stream == nil {
		req.Msg = "流记录不存在"
		ctx.JSON(req)
		return
	}

	pusher := client.GetServer().GetPusher(stream.Source)
	if pusher == nil {
		pusher = client.NewPusher(stream.Key, stream.Source, stream.RoomName)
	}

	if pusher != nil {
		log.Debugln("room_name:", stream.RoomName, "key:", stream.Key, "room_id:", stream.ID)
		client.GetServer().AddPusher(pusher)
		ctx.JSON(req)
		return
	}

	req.Code = 40000
	req.Msg = "启动失败"
	ctx.JSON(req)
}

func stop(ctx iris.Context) {
	req := Req{nil, 20000, "停止拉流"}

	id := ctx.Params().GetUintDefault("id", 0)
	stream, err := models.StopStream(id)
	if err != nil {
		req.Msg = err.Error()
		ctx.JSON(req)
		return
	}

	if stream == nil {
		req.Msg = "流记录不存在"
		ctx.JSON(req)
		return
	}

	pusher := client.GetServer().GetPusher(stream.Source)
	if pusher == nil {
		req.Msg = "停止失败"
		ctx.JSON(req)
		return
	}

	log.Debugln("room_name:", stream.RoomName, "key:", stream.Key, "room_id:", stream.ID)
	client.GetServer().RemovePusher(pusher)
	ctx.JSON(req)

}

func delete(ctx iris.Context) {
	req := Req{nil, 20000, "删除成功"}

	id := ctx.Params().GetUintDefault("id", 0)
	err := models.DeleteStream(id)
	if err != nil {
		req.Msg = err.Error()
		ctx.JSON(req)
		return
	}

	ctx.JSON(req)
}
