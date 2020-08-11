package restore

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	helmclient "github.com/joelanford/helm-operator/pkg/client"
	"github.com/joelanford/helm-operator/pkg/hook"
	"github.com/prometheus/common/log"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Restore struct {
	hook.PostHookFunc
	client client.Client
	acg    helmclient.ActionClientGetter
}

const (
	//VeleroDir is the relative directory where snapshot chart is loaded
	VeleroDir string = "Velero"
)

func NewRestore(client client.Client, acg helmclient.ActionClientGetter) Restore {
	return Restore{
		client: client,
		acg:    acg,
		//exec:   BckupPostHook,
	}
}

//Restore Posthook : upgrade to no postgresql, restore cr
func (r *Restore) RestorePostHook(obj *unstructured.Unstructured, rel release.Release, log logr.Logger) error {

	log.Info("IN restore HOOK")

	var vals chartutil.Values = rel.Chart.Values
	restoreEnabled, err := vals.PathValue("restore.enabled")
	if err != nil || restoreEnabled.(bool) == false {
		log.Info("RESTORE DISABLED")
		return nil
	}

	//validate before restore
	if !ValidateValues(rel.Chart.Values) {
		err := errors.New("backup and restore cannot be enabled simultaneously")
		log.Error(err, "please choose a single operation")
		return err
	}

	acf, err := r.acg.ActionClientFor(obj)
	if err != nil {
		log.Error(err, "unable to create acf")
		return err
	}
	postgresEnabled, err := vals.PathValue("postgres.enabled")
	if err == nil && postgresEnabled.(bool) == true {
		postgresMap := map[string]interface{}{
			"postgresql": map[string]interface{}{
				"enabled": false,
			},
		}
		semrel, err := acf.Upgrade(rel.Name, rel.Namespace, rel.Chart, postgresMap)
		if err != nil {
			log.Error(err, "failed to update matrix")
			return err
		}
		log.Info(fmt.Sprintf("release: %+v", semrel.Info))
	}
	u, err := createRestoreCR(rel.Chart.Values, log)
	if err != nil {
		return err
	}

	err = r.client.Create(context.Background(), u)
	if err != nil {
		log.Error(err, "unable to create Restore CR")
		return err
	}
	u1 := &unstructured.Unstructured{}
	u1.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Kind:    "Restore",
		Version: "v1",
	})

	for {
		r.client.Get(context.TODO(), client.ObjectKey{
			Namespace: "default",
			Name:      "matrix-restore",
		}, u1)
		//log.Info(fmt.Sprintf("restore: %v", u1))

		/*for k := range u1.Object {
			log.Info(fmt.Sprintf("keys: %+v", k))
		}*/

		//log.Info(fmt.Sprintf("status: %v", u1.Object["status"]))
		if u1.Object["status"] != nil {
			status := u1.Object["status"].(map[string]interface{})
			//log.Info(fmt.Sprintf("phase: %v", status["phase"]))
			if status["phase"] == "Completed" {
				log.Info("Restore Completed. Cleaning up Restore CR")
				r.client.Delete(context.TODO(), u1)
				break
			}
		}
	}
	//} // if restore enabled

	//}
	return nil

}

func createRestoreCR(vals chartutil.Values, log logr.Logger) (*unstructured.Unstructured, error) {

	//v := vals.AsMap()

	//log.Info(fmt.Sprintf("created %+v", v))

	backupName, err := vals.PathValue("restore.backupName")
	if err != nil || backupName == "" {
		log.Error(err, "Please provide a valid restore.backupName in the CR")
		return nil, err
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Kind:    "Restore",
		Version: "v1",
	})

	u.Object = map[string]interface{}{
		"apiVersion": "velero.io/v1",
		"kind":       "Restore",
		"metadata": map[string]interface{}{
			"name":      "matrix-restore",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"backupName": backupName.(string),
			"excludedResources": []interface{}{
				"nodes",
				"events",
				"events.events.k8s.io",
				"backups.velero.io",
				"restores.velero.io",
				"resticrepositories.velero.io",
			},
			"restorePVs": true,
		},
	}

	return u, nil
}

func ValidateValues(vals chartutil.Values) bool {

	restore, err := vals.PathValue("restore.enabled")
	if err != nil {
		return true
	}
	log.Info(fmt.Sprintf("restore e: %v", restore.(bool)))
	backup, err := vals.PathValue("backup.enabled")
	if err != nil {
		return true
	}
	log.Info(fmt.Sprintf("backup e: %v", backup.(bool)))
	if restore.(bool) == true && backup.(bool) == true {
		return false
	}

	return true

}

//TODO: Remove comment blob on cleanup
/*
//RestorePostHook restores matrix to a previously backed release
func (r *Restore) RestorePreHook(obj *unstructured.Unstructured, val chartutil.Values, log logr.Logger) error {

	log.Info("IN restore HOOK")

	acf, err := r.acg.ActionClientFor(obj)
	if err != nil {
		log.Error(err, "unable to create acf")
		return err
	}

	acf.Upgrade()


	_, err = launchVelero(val, acf, log)
	if err != nil {
		//log.Error(err, "Failed to load velero chart")
		return err
	}*/
/*
			u := createRestoreCR(val, log)

			err := r.client.Create(context.Background(), u)
			if err != nil {
				log.Error(err, "unable to create 37")
			}


		log.Info(fmt.Sprintf("new restore: %v", u))

	* /
	return nil
}


/*
//Load Chart
func launchVelero(vals chartutil.Values, acf helmclient.ActionInterface, log logr.Logger) (*release.Release, error) {
	//log.Info(fmt.Sprintf("values: %+v", velero))
	v := vals.AsMap()
	//log.Info(fmt.Sprintf("velero values: %+v", v))

	for k := range v {
		log.Info(fmt.Sprintf("keys: %+v", k))
	}

	log.Info(fmt.Sprintf("velero: %+v", v["velero"]))

	//acf.Install("velero-restore", "")

	vMap := v["velero"]
	log.Info(fmt.Sprintf("type: %+v", reflect.TypeOf(vMap)))


	velMap := vMap.(map[string]interface{})

	//config := velMap["configuration"]
	//credentials := velMap["credentials"]
	//log.Info(fmt.Sprintf("config: %+v", config))

	//chart, err := loadChart(log)
	chart, err := LoadSnapshotChart(log)
	if err != nil {
		log.Error(err, "Failed to load velero chart")
		return nil, err
	}
	log.Info(fmt.Sprintf("chart: %+v", chart.Name()))
	log.Info(fmt.Sprintf("chart-metadata: %+v", chart.Metadata))

	/*
		annotation := make(map[string]string)

		annotation["meta.helm.sh/release-name"] = "velero"
		annotation["meta.helm.sh/release-namespace"] = "default"
		chart.Metadata.Annotations = annotation

		log.Info(fmt.Sprintf("annotation : %+v", chart.Metadata.Annotations))
	*

	/*rel, err := acf.Install("velero", "", chart, velMap)

	if err != nil {
		log.Error(err, "Failed to install velero chart")
		return nil, err
	}
	*
	settings := helmcli.New()

	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), lg.Printf); err != nil {
		return nil, err
	}

	installer := action.NewInstall(actionConfig)
	//installer.Namespace = "default"
	installer.ReleaseName = "velero"

	rel, err := installer.Run(chart, velMap)
	if err != nil {
		log.Error(err, "failed to install velero")
		return nil, err
	}

	log.Info(fmt.Sprintf("release: %+v", rel))

	return nil, nil

}


func loadChart(log logr.Logger) (*chart.Chart, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	//var HelmChartsDir string = "helm-charts/matrix"
	//var dependencies string = "charts"
	var chartPath string = "helm-charts/matrix/charts/velero-2.12.0.tgz"
	velChartPath := filepath.Join(projectDir, chartPath)

	chart, err := loader.Load(velChartPath)
	//log.Info(fmt.Sprintf("chart: %+v", chart.Files))

	return chart, nil
}


//Load Snapshot Chart
func LoadSnapshotChart(log logr.Logger) (*chart.Chart, error) {

	log.Info("Attempt to load snapshot chart")

	settings := helmcli.New()
	getters := getter.All(settings)
	c := downloader.ChartDownloader{
		Out:              os.Stderr,
		Getters:          getters,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	chartURL, err := repo.FindChartInRepoURL("https://vmware-tanzu.github.io/helm-charts", "velero", "2.12.0", "", "", "", getters)
	if err != nil {
		return nil, err
	}

	tmpDir, err := ioutil.TempDir("", "velero-helm-chart")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			//log.Errorf("Failed to remove temporary directory %s: %s", tmpDir, err)
			log.Error(err, fmt.Sprintf("Failed to remove temporary directory %s: %s", tmpDir, err))
		}
	}()

	chartArchive, _, err := c.DownloadTo(chartURL, "2.12.0", tmpDir)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(chartArchive)
	if err != nil {
		return nil, err
	}

	//save chart to snapshot dir
	if err := chartutil.SaveDir(chart, SnapshotDir); err != nil {
		return nil, err
	}

	log.Info("loaded velero chart")

	return chart, nil

}

const (
	//SnapshotDir is the relative directory where snapshot chart is loaded
	SnapshotDir string = "velero"
)
*/
