package snapshot

import (
	"fmt"
	"io/ioutil"
	lg "log"
	"os"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	//SnapshotDir is the relative directory where snapshot chart is loaded
	SnapshotDir string = "velero"
)

//SnapPreHook helps to take snapshot at deletion
func SnapPreHook(obj *unstructured.Unstructured, vals chartutil.Values, log logr.Logger) error {

	log.Info("IN SNAPSHOT HOOK")

	chart, err := loadSnapshotChart(log)
	if err != nil {
		log.Error(err, "Failed to load velero chart")
		return err
	}

	rel, err := installChart(chart)

	if err != nil {
		log.Error(err, "Failed to install velero chart")
		return err
	}

	log.Info(fmt.Sprintf("release info: %v", rel))

	log.Info("OUT SNAPSHOT HOOK")

	return nil
}

func loadSnapshotChart(log logr.Logger) (*chart.Chart, error) {

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

	log.Info("loaded snapshot chart")

	return chart, nil

}

func installChart(chart *chart.Chart) (*release.Release, error) {

	settings := helmcli.New()

	//a := action.Configuration.
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), lg.Printf); err != nil {
		return nil, err
	}

	installer := action.NewInstall(actionConfig)
	installer.Namespace = "default"
	installer.ReleaseName = "velero-snapshotter"

	//release
	//values map
	rel, err := installer.Run(chart, nil)
	if err != nil {
		return nil, err
	}
	return rel, nil
}
