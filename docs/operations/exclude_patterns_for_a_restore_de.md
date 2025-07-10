# Ausschließen von Ressourcen während eines Restores

Das Restore ist in zwei Schritte unterteilt. Als Erstes wird ein Cleanup durchgeführt. Dieses Cleanup löscht alle 
Ressourcen des Clusters, die nicht für das Backup benötigt werden. Alle Ressourcen, die der Backup-Stack benötigt, 
besitzen die Annotation `k8s.cloudogu.com/part-of: backup`. 

Im zweiten Schritt wird der Backup-Provider verwendet, um ein Restore durchzuführen und alle Ressourcen aus dem 
Backup dem Cluster hinzuzufügen.

## Dateien im Cleanup ausschließen

Mit dem Attribute `cleanup.exclude` in der `values.yaml` lassen sich beliebige Ressourcen aus dem Cleanup ausschließen.
Die Ressourcen müssen lediglich im GVKN-Pattern (Group, Version, Kind, Name) angegeben werden. 
```yaml
...
cleanup:
  exclude:
    - name: "k8s-backup-operator"
      kind: "Component"
      version: "*"
      group: "k8s.cloudogu.com"
    - name: "test-certificate"
      kind: "Secret"
      version: "*"
...
```
Standardmäßig werden alle Ressourcen, die für das Backup benötigt werden, der Ces-Loadbalancer und das Zertifikat beim Cleanup 
ausgeschlossen. Diese Ressourcen bleiben nach dem Cleanup erhalten.

## Dateien im Restore-Prozess ausschließen

Für den Restore-Provider `velero` existiert ein 
[Plugin zum Ausschließen von Ressourcen aus dem Backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/). 
Dieses Plugin kann mit `velero` im Cluster angewendet und genutzt werden, um Ressourcen während des 
Restore-Prozesses auszuschließen. Für weitere Informationen siehe [hier](https://github.com/cloudogu/k8s-velero/blob/develop/docs/exclude_out_of_restore_de.md)

## Zusammenspiel

Diese beiden Ausschließungsprozesse sollten zusammen verwendet werden. Dadurch ist es möglich Ressourcen
vor dem Einspielen eines Restores im Cluster zu behalten und im Nachhinein weiter zu verwenden.