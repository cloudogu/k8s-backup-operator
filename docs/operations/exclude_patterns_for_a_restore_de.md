# Ausschließen von Ressourcen während eines Restores

Das Restore ist in zwei Schritte unterteilt. Als Erstes wird ein Cleanup durchgeführt. Dieses Cleanup löscht alle Dogus.

Im zweiten Schritt wird der Backup-Provider verwendet, um ein Restore durchzuführen und alle Dogus aus dem Backup dem 
Cluster hinzuzufügen.

## Ressourcen im Restore-Prozess ausschließen

Für den Restore-Provider `velero` existiert ein 
[Plugin zum Ausschließen von Ressourcen aus dem Backup](https://github.com/cloudogu/velero-plugin-for-restore-exclude/). 
Dieses Plugin kann mit `velero` im Cluster angewendet und genutzt werden, um Ressourcen während des 
Restore-Prozesses auszuschließen. Für weitere Informationen siehe [hier](https://github.com/cloudogu/k8s-velero/blob/develop/docs/exclude_out_of_restore_de.md)
