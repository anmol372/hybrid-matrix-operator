package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/hook"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Restore struct {
	hook.PostHookFunc
	client client.Client
}

func NewRestore(client client.Client) Restore {
	return Restore{
		client: client,
		//exec:   BckupPostHook,
	}
}

//RestorePostHook restores matrix to a previously backed release
func (r *Restore) RestorePostHook(obj *unstructured.Unstructured, rel release.Release, log logr.Logger) error {

	log.Info("IN restore HOOK")

	u := createRestoreCR(rel.Chart.Values, log)

	err := r.client.Create(context.Background(), u)
	if err != nil {
		log.Error(err, "unable to create 37")
	}

	log.Info(fmt.Sprintf("new restore: %v", u))
	return nil
}

func createRestoreCR(vals chartutil.Values, log logr.Logger) *unstructured.Unstructured {

	v := vals.AsMap()

	log.Info(fmt.Sprintf("created %+v", v))

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Kind:    "Backup",
		Version: "v1",
	})

	u.Object = map[string]interface{}{
		"apiVersion": "velero.io/v1",
		"kind":       "Restore",
		"metadata": map[string]interface{}{
			"name":      "matrix-" + time.Now().Format("20060102150405") + "-restore",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"backupName": "matrix-20200727211715-backup",
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

	return u
}
