# Backup-Prozess

Ein CES-Backup beschreibt einen Sicherungslauf. Für einen neuen Lauf ermittelt der Backup-Controller die benötigten
Backup-Units und erstellt für jede Unit ein Velero-Backup. `ProviderSucceeded` fasst die Ergebnisse der Units
zusammen. Der [Backup-Umfang](backup_scope_de.md) beschreibt, welche Ressourcen zu den Units gehören.

Der Velero-Synchronisationscontroller importiert ältere Velero-Backups als gleichnamige CES-Backup-Ressourcen.
Mehr zum Abgleich nach einem Restore steht unter [Backup-Synchronisierung](backup_sync_de.md).

## Ablauf eines lokalen Backups

Der Controller prüft zuerst die Voraussetzungen. Dann übernimmt er das
[gemeinsame Backup-/Restore-Lease](parallel_access_de.md) des Namespace und aktiviert den Wartungsmodus. Er legt
fehlende Velero-Backups an und prüft ihren Fortschritt bei weiteren Reconciles. Reine Statusänderungen lösen im
Backup-Controller keinen weiteren Reconcile aus. Solange Provider-Arbeit aussteht, plant der Controller deshalb einen
erneuten Durchlauf ein.

`ProviderSucceeded` beschreibt das Ergebnis bei Velero. `Succeeded` beschreibt den Abschluss des CES-Backup-Laufs. Dazwischen
deaktiviert der Controller den Wartungsmodus und gibt anschließend das Lease frei. Der Wartungsmodus gehört dem
Lease-Inhaber. Schlägt die Deaktivierung fehl, bleibt der Lauf in der Abschlussphase.

## Abbruch und Löschung

Ein abgelaufenes Zeitfenster (`retryTimeLimit`) kann den CES-Backup-Lauf beenden, während Velero noch arbeitet.
Der Controller gibt Wartungsmodus und Lease frei, damit der abgebrochene Lauf das System nicht weiter blockiert.
Die CES-Backup-CR bleibt als Fehlerhistorie erhalten. Velero-Backups, die zum Zeitpunkt des Abbruchs noch liefen werden
gelöscht. Dies soll verhindern, dass Daten wiederhergestellt werden, die Velero möglicherweise nach Ende des
Wartungsmodus gesichert hat.

Eine Löschanforderung an der CES-Backup-CR hat vor dem normalen Ablauf Vorrang. Zugehörige Velero-Backups werden gelöscht.

## Einstieg im Code

- Der [Backup-Controller](../../internal/controller/backup/controller.go) wählt den Reconcile-Pfad und steuert
  Wiederholungen.
- Die [Erzeugung der Velero-Backups](../../internal/controller/backup/unit_backups.go) legt fest, wie jede Unit
  gesichert wird. Die [Auswertung der Units](../../internal/controller/backup/unit_completion.go) fasst ihre
  Ergebnisse zusammen.
- Der [Reconciler](../../internal/controller/backup/reconciler.go) enthält die Prüfungen der Voraussetzungen, die
  Steuerung des Wartungsmodus und die Abbruch- und Löschpfade.
- Der [Velero-Synchronisationscontroller](../../internal/controller/backup/velero_backup_controller.go) entscheidet,
  welche Velero-Backups in den Abgleich eingehen.
