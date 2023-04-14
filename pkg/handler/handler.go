package handler

import (
	"github.com/nais/bifrost/pkg/config"
	"github.com/sirupsen/logrus"
	admin "google.golang.org/api/sqladmin/v1beta4"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	kubeClient   ctrl.Client
	googleClient *admin.Service
	config       *config.Config
	sqlInstance  *admin.DatabaseInstance
	logger       *logrus.Logger
}

func NewHandler(kubeClient ctrl.Client, googleClient *admin.Service, config *config.Config, sqlInstance *admin.DatabaseInstance, logger *logrus.Logger) *Handler {
	return &Handler{
		kubeClient:   kubeClient,
		googleClient: googleClient,
		config:       config,
		sqlInstance:  sqlInstance,
		logger:       logger,
	}
}
