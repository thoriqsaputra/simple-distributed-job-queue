package _dataloader

import (
	"context"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"

	"github.com/graph-gophers/dataloader/v6"
	"github.com/labstack/echo/v4"
)

// GeneralDataloader holds the dataloaders for various entities.
type GeneralDataloader struct {
	JobLoader *dataloader.Loader
	jobRepo   _interface.JobRepository
}

// NewGeneralDataloader creates and returns a new instance of GeneralDataloader.
// It takes the JobRepository as a dependency to initialize its loaders.
func NewGeneralDataloader(repo _interface.JobRepository) *GeneralDataloader {
	dl := &GeneralDataloader{
		jobRepo: repo,
	}
	// Initialize the JobLoader here, passing the batch function from this dataloader instance.
	dl.JobLoader = dataloader.NewBatchedLoader(dl.JobBatchFunc, dataloader.WithCache(dataloader.NewCache()))
	return dl
}

// EchoMiddelware is an Echo middleware that adds the GeneralDataloader to the request context.
func (g *GeneralDataloader) EchoMiddelware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// add general dataloader into echo context
		oriReq := c.Request()
		req := oriReq.WithContext(context.WithValue(oriReq.Context(), constant.DataloaderContextKey, g))
		c.SetRequest(req)
		return next(c)
	}
}