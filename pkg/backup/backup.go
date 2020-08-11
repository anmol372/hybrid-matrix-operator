package backup

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	helmclient "github.com/joelanford/helm-operator/pkg/client"
	"github.com/joelanford/helm-operator/pkg/hook"
	"github.com/joelanford/helm-operator/pkg/restore"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backup struct {
	hook.PostHookFunc
	client client.Client
	acg    helmclient.ActionClientGetter
}

func NewBackup(client client.Client, acg helmclient.ActionClientGetter) Backup { //, f hook.PostHookFunc
	return Backup{
		client: client,
		acg:    acg,
	}
}

//BckupPostHook deploys a kind:Backup
//1. Instant Backup
func (b *Backup) BckupPostHook(obj *unstructured.Unstructured, rel release.Release, log logr.Logger) error {

	log.Info("IN Backup HOOK 123")

	var vals chartutil.Values = rel.Chart.Values
	backupEnabled, err := vals.PathValue("backup.enabled")
	if err != nil || backupEnabled.(bool) == false {
		log.Info(fmt.Sprintf("backupenabled: %v", backupEnabled.(bool)))
		log.Error(err, "backupEnabled value")
		return nil
	}
	log.Info(fmt.Sprintf("backupenabled: %v", backupEnabled.(bool)))
	//validate before backup
	if !restore.ValidateValues(rel.Chart.Values) {
		err := errors.New("backup and restore cannot be enabled simultaneously")
		log.Error(err, "please choose a single operation")
		return err
	}

	u, _ := createBackupCR(rel, log)

	err = b.client.Create(context.Background(), u)
	if err != nil {
		log.Error(err, "unable to create backup CR")
		return err
	}
	log.Info(fmt.Sprintf("backup: %v", u))
	u1 := &unstructured.Unstructured{}
	u1.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "velero.io",
		Kind:    "Backup",
		Version: "v1",
	})

	for {
		b.client.Get(context.TODO(), client.ObjectKey{
			Namespace: "default",
			Name:      u.GetName(),
		}, u1)

		log.Info(fmt.Sprintf("backup: %v", u))

		if u1.Object["status"] != nil {
			status := u1.Object["status"].(map[string]interface{})
			if status["phase"] == "Completed" {
				log.Info("Backup Completed. Updating release")
				acf, err := b.acg.ActionClientFor(obj)
				if err != nil {
					log.Error(err, "unable to create acf")
					return err
				}
				backupMap := map[string]interface{}{
					"backup": map[string]interface{}{
						"enabled": false,
					},
				}
				//update crd
				semrel, err := acf.Upgrade(rel.Name, rel.Namespace, rel.Chart, backupMap)
				if err != nil {
					log.Error(err, "failed to update matrix")
					return err
				}
				log.Info(fmt.Sprintf("release: %+v", semrel.Info))
				break
			}
			//}
			/*if status["phase"] == "PartiallyFailed" {
				log.Info("FAILED FAILED FAILED FAILED")
				return nil
			}*/
		}

	}

	return nil
}

func createBackupCR(rel release.Release, log logr.Logger) (*unstructured.Unstructured, error) {

	var vals chartutil.Values = rel.Chart.Values
	backupName, err := vals.PathValue("backup.backupName")
	if err != nil || backupName.(string) == "" {
		backupName = "matrix-backup-" + rel.Name + "-" + strconv.Itoa(rel.Version)
	}

	log.Info(fmt.Sprintf("backupName: %v", backupName))

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
			"name":      backupName.(string),
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

	return u, nil
}
