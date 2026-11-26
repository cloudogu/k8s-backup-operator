# Ausschließen von Ressourcen während eines Restores

Das Restore ist in zwei Schritte unterteilt. Als Erstes wird ein Cleanup durchgeführt. Dieses Cleanup löscht alle Dogus, die wiederhergestellt werden.
Bei einem Restore aus Backup-Units bleiben Dogus und Components, die über `excludedDogus`, `excludedComponents` oder
`options.ignoreV2Dogus` ausgeschlossen sind, erhalten. Bei einem Restore aus einem Legacy-Backup werden alle Dogus gelöscht.

Im zweiten Schritt wird der Backup-Provider verwendet, um ein Restore durchzuführen und alle Dogus aus dem Backup dem 
Cluster hinzuzufügen.
