package k8s

import (
	"context"
	"sync/atomic"

	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
	crdv1alpha1 "github.com/sentinel-group/sentinel-go-datasource-opensergo-poc/api/v1alpha1"
	"github.com/sentinel-group/sentinel-go-datasource-opensergo-poc/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = crdv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type CRDType int32

const (
	FaultToleranceRuleCRDType CRDType = iota
	RateLimitStrategyCRDType
)

func (c CRDType) String() string {
	switch c {
	case FaultToleranceRuleCRDType:
		return "fault-tolerance.opensergo.io/FaultToleranceRule"
	case RateLimitStrategyCRDType:
		return "fault-tolerance.opensergo.io/RateLimitStrategy"
	default:
		return "Undefined"
	}
}

type DataSource struct {
	crdManager  ctrl.Manager
	controllers map[CRDType]reconcile.Reconciler

	namespace string
	appName   string

	ctx       context.Context
	ctxCancel context.CancelFunc
	started   atomic.Value

	faultToleranceRuleAggregator *controllers.FaultToleranceRuleAggregator
}

// NewDataSource creates a OpenSergo data-source with given Kubernetes namespace.
// All Controllers take effective only with matched namespace.
func NewDataSource(namespace string, app string) (*DataSource, error) {
	ctrl.SetLogger(&k8SLogger{
		l:             logging.GetGlobalLogger(),
		level:         logging.GetGlobalLoggerLevel(),
		names:         make([]string, 0),
		keysAndValues: make([]interface{}, 0),
	})
	k8sConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	mgr, err := ctrl.NewManager(k8sConfig, ctrl.Options{
		Scheme: scheme,
		// disable metric server
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	k := &DataSource{
		crdManager:                   mgr,
		controllers:                  make(map[CRDType]reconcile.Reconciler, 4),
		namespace:                    namespace,
		appName:                      app,
		ctx:                          ctx,
		ctxCancel:                    cancel,
		faultToleranceRuleAggregator: controllers.NewRuleAggregator(),
	}
	return k, nil
}

func (k *DataSource) RegisterControllersAndStart() error {
	err := k.RegisterController(FaultToleranceRuleCRDType)
	if err != nil {
		return err
	}
	err = k.RegisterController(RateLimitStrategyCRDType)
	if err != nil {
		return err
	}
	return k.Run()
}

// RegisterController registers given CRD type and CRD name.
// For each CRD type, it can be registered only once.
func (k *DataSource) RegisterController(crd CRDType) error {
	_, exist := k.controllers[crd]
	if exist {
		return errors.Errorf("duplicated register crd for %s", crd.String())
	}

	switch crd {
	case FaultToleranceRuleCRDType:
		controller := controllers.NewFaultToleranceRuleReconciler(k.crdManager, k.namespace, k.appName, k.faultToleranceRuleAggregator)
		err := controller.SetupWithManager(k.crdManager)
		if err != nil {
			return err
		}
		k.controllers[FaultToleranceRuleCRDType] = controller
		setupLog.Info("OpenSergo FaultToleranceRule CRD controller has been registered successfully")
		return nil
	case RateLimitStrategyCRDType:
		controller := controllers.NewRateLimitStrategyReconciler(k.crdManager, k.namespace, k.appName, k.faultToleranceRuleAggregator)
		err := controller.SetupWithManager(k.crdManager)
		if err != nil {
			return err
		}
		k.controllers[RateLimitStrategyCRDType] = controller
		setupLog.Info("OpenSergo RateLimitStrategy CRD controller has been registered successfully")
		return nil
	default:
		return errors.Errorf("unsupported CRDType: %d", int(crd))
	}
}

// Close exit the K8S DataSource
func (k *DataSource) Close() error {
	k.ctxCancel()
	return nil
}

// Run runs the k8s DataSource
func (k *DataSource) Run() error {

	// +kubebuilder:scaffold:builder
	go util.RunWithRecover(func() {
		setupLog.Info("Starting OpenSergo CRD manager for Sentinel")
		if err := k.crdManager.Start(k.ctx); err != nil {
			setupLog.Error(err, "problem running OpenSergo CRD manager for Sentinel")
		}
		setupLog.Info("Sentinel OpenSergo CRD data-source closed")
	})
	return nil
}
