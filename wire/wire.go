//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/wibus-wee/synclax/pkg"
	"github.com/wibus-wee/synclax/pkg/asynctask"
	"github.com/wibus-wee/synclax/pkg/config"
	"github.com/wibus-wee/synclax/pkg/handler"
	symphonyctl "github.com/wibus-wee/synclax/pkg/symphony/control"
	"github.com/wibus-wee/synclax/pkg/zcore/app"
	"github.com/wibus-wee/synclax/pkg/zcore/injection"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
	"github.com/wibus-wee/synclax/pkg/zgen/taskgen"

	"github.com/google/wire"
)

func InitApp() (*app.App, error) {
	wire.Build(
		injection.InjectAuth,
		injection.InjectTaskStore,
		handler.NewHandler,
		handler.NewValidator,
		symphonyctl.NewManager,
		taskgen.NewTaskHandler,
		taskgen.NewTaskRunner,
		asynctask.NewExecutor,
		model.NewModel,
		config.NewConfig,
		pkg.ProvidePluginMeta,
		pkg.Init,
		pkg.InitAnclaxApplication,
		app.NewPlugin,
	)
	return nil, nil
}
