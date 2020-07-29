package backup

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	helmclient "github.com/joelanford/helm-operator/pkg/client"
	"github.com/joelanford/helm-operator/pkg/hook"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backup struct {
	hook.PostHookFunc
	client        client.Client
	acg           helmclient.ActionClientGetter
	createdBackup bool
}

func NewBackup(client client.Client, acg helmclient.ActionClientGetter) Backup { //, f hook.PostHookFunc
	return Backup{
		client:        client,
		acg:           acg,
		createdBackup: false,
		//exec:   BckupPostHook,
	}
}

//1. Instant Backup

//BckupPostHook deploys a kind:Backup
func (b *Backup) BckupPostHook(obj *unstructured.Unstructured, rel release.Release, log logr.Logger) error {

	log.Info("IN Backup HOOK")

	log.Info(fmt.Sprintf("labels:\t%+v", obj.GetLabels()))

	rel1 := rel.Info.FirstDeployed
	rel2 := rel.Info.LastDeployed

	/*if rel1.Before(rel2) {

	}*/

	ver := rel.Version
	if ver == 1 && b.createdBackup == false {

		log.Info(fmt.Sprintf("first: %v, last: %v", rel1, rel2))

		ver2 := rel.Chart.Metadata.Version

		log.Info(fmt.Sprintf("rel v: %v, semv2: %v", ver, ver2))

		u := createBackupCR(rel.Chart.Values, log)

		err := b.client.Create(context.Background(), u)
		if err != nil {
			log.Error(err, "unable to create 63")
		}
		log.Info(fmt.Sprintf("new backup: %v", u))

		//log.Info(fmt.Sprintf("created %v", u))

		b.createdBackup = true

	}
	return nil
}

func createBackupCR(vals chartutil.Values, log logr.Logger) *unstructured.Unstructured {

	v := vals.AsMap()

	log.Info(fmt.Sprintf("created %+v", v))

	enabled, _ := v["backupsEnabled"]

	//velero, _ := v["velero"]
	en, _ := vals.PathValue("velero.enabled")

	log.Info(fmt.Sprintf("backup enabled: %v", enabled))

	log.Info(fmt.Sprintf("velero enabled: %v", en))

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Kind:    "Backup",
		Version: "v1",
	})

	u.Object = map[string]interface{}{
		"apiVersion": "velero.io/v1",
		"kind":       "Backup",
		"metadata": map[string]interface{}{
			"name":      "matrix-" + time.Now().Format("20060102150405") + "-backup",
			"namespace": "default",
			"labels": map[string]interface{}{
				"velero.io/storage-location": "default",
			},
		},
		"spec": map[string]interface{}{
			"hooks": map[string]interface{}{},
			"includeNamespaces": []interface{}{
				"default",
			},
			"storageLocation": "matrix-backup",
			"ttl":             "720h0m0s",
		},
	}

	return u
}
