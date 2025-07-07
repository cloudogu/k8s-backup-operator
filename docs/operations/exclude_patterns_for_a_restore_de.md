# Ausschließen von Dateien während eines Restores

Das Restore ist in zwei Schritte unterteilt. Als Erstes wird ein Cleanup durchgeführt. Dieses Cleanup löscht alle 
Ressourcen des Clusters, die nicht für das Backup benötigt werden. Alle Ressourcen, die der Backup-Stack benötigt, 
besitzen die Annotation `k8s.cloudogu.com/part-of: backup`. 

Im zweiten Schritt wird der Backup-Provider verwendet, um ein Restore durchzuführen und alle Ressourcen aus dem 
Backup dem Cluster hinzuzufügen.

## Dateien im Cleanup ausschließen

Mit dem Attribute `cleanup.exclude` in der `values.yaml` lassen sich beliebige Ressourcen aus dem Cleanup ausschließen.
Die Ressourcen müssen lediglich im GVKN-Pattern (Group, Version, Kind, Name) angegeben werden. Standardmäßig werden 
alle Ressourcen, die für das Backup benötigt werden, der Ces-Loadbalancer und das Zertifikat beim Cleanup 
ausgeschlossen. Diese Ressourcen bleiben nach dem Cleanup erhalten.

## Dateien im Restore-Prozess ausschließen

Für den Restore-Provider `velero` existiert ein 
[Plugin zum Ausschließen von Ressourcen aus dem Backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/),
welches eingespielt wird. Dieses Plugin kann mit `velero` im Cluster angewendet und mit dem Attribute 
`excludedFromRestores` konfiguriert werden. Dabei gibt es die gleichen Konfigurationsmöglichkeiten, wie bei dem 
Ausschließen bei Cleanup über das GVKN-Pattern. Ressourcen, die hier ausgeschlossen werden und in dem Backup 
vorhanden sind, das eingespielt werden soll, werden bei diesem Restore ignoriert.

## Zusammenspiel

Diese beiden Ausschließungsprozesse sollten zusammen verwendet werden. Dadurch ist es möglich Ressourcen, die sich 
vor dem Einspielen eines Restores im Cluster zu behalten und im Nachhinein zu verwenden.