# Aufsetzen eines Backup-Buckets inkl. Zugriff in der Google Cloud

## Aufsetzen mit Terraform

Zum Aufsetzen eines Google Cloud Buckets mit Terraform ist 
[hier](https://github.com/cloudogu/k8s-ecosystem/blob/develop/terraform/examples/ces_google_gke/google_bucket/README.md)
ein Beispiel zu finden.

## Aufsetzen mit Google Cloud Benutzeroberfläche

### Voraussetzungen
- Ein Service Account / Dienstkonto muss vorhanden sein

Vorraussetzungen für eine Verschlüsselung des Buckets:
- Im Projekt muss die Cloud Key Management API aktiviert sein, wenn Verschlüsselung erwünscht
- Der Service Account (bzw. das Dienstkonto) muss die Rolle "cloudkms.cryptoKeyEncrypterDecrypter" besitzen
- Key zum Verschlüsseln
- die Verschlüsselung ist optional

#### Cloud Key Management Service API aktivieren
- KMS-Bereich des Projekts aufrufen
- Cloud Key Management Service API aktivieren

#### Service Account erstellen
- IAM-Bereich des Projekts aufrufen
- Service Account / Dienstkonto erstellen
- Name erstellen
- Beschreibung einstellen
- Anlegen
- Rolle "cloudkms.cryptoKeyEncrypterDecrypter" zuweisen (optional, für Verschlüsselung nötig)

#### Key zum Verschlüsseln erstellen (optional)
- Nach Security -> Key management -> Key rings wechseln
- Neuen Key ring anlegen, falls nötig
    - Region: europe-west3
- Key anlegen
    - Name vergeben
    - Generated Key
    - Symmetric encrypt/decrypt
    - Key Rotation: Never (oder besseres Vorgehen etablieren)
    - Labels
        - purpose: (wofür, soll dieser Key verwendet werden)
        - key-name: $NAME (wie oben)
        - team: $TEAM (bspw. ces)

### Erstellen des Buckets
- Buckets im Cloud-Storage-Bereich des Projekts aufrufen
- "Create bucket" klicken
- Name vergeben
- Labels vergeben
    - team: $TEAM (bspw. ces)
    - purpose: $PURPOSE (wofür, soll dieser Bucket verwendet werden)
    - bucket_name: $NAME (wie oben)
- Location type: Region europe-west3
- Storage class: Standard
    - Oder Nearline/Coldline, wenn es selten Zugriff gibt
- Prevent public access: Fine-grained
- Data protection: **Kein** soft-delete!
- Data encryption: Cloud KMS key
    - Oben angelegten Key auswählen

### Serviceaccount auf den Bucket berechtigen
Diese Schritte müssen von einem Google-Cloud-Administrator durchgeführt werden.

- Email des Service Accounts auslesen

```
SERVICE_ACCOUNT_EMAIL=$(gcloud iam service-accounts list --project $PROJECT_ID --filter="displayName:Velero service account" --format 'value(email)')
```
  - oder in der Dienstkonten-Seite in der Google Cloud

- Berechtigungen zum Serviceaccount hinzufügen
```bash
ROLE_PERMISSIONS=(
    compute.disks.get
    compute.disks.create
    compute.disks.createSnapshot
    compute.projects.get
    compute.snapshots.get
    compute.snapshots.create
    compute.snapshots.useReadOnly
    compute.snapshots.delete
    compute.zones.get
    storage.objects.create
    storage.objects.delete
    storage.objects.get
    storage.objects.list
    iam.serviceAccounts.signBlob
)
```
  - diese Berechtigungen werden z.B. in der Rolle velero.server.2 zusammengefasst
    - sonst: Rolle erstellen

```
gcloud iam roles create velero.server --project $PROJECT_ID --title "Velero Server" --permissions "$(IFS=","; echo "${ROLE_PERMISSIONS[*]}")"```
```
- Rolle dem Serviceaccount zuweisen

```
gcloud projects add-iam-policy-binding $PROJECT_ID --member serviceAccount:$SERVICE_ACCOUNT_EMAIL --role projects/$PROJECT_ID/roles/velero.server
```

- Serviceaccount mit Bucket verknüpfen

```
gsutil iam ch serviceAccount:$SERVICE_ACCOUNT_EMAIL:objectAdmin gs://${BUCKET_NAME}
```

### Zugriff aus einem anderen Projekt konfigurieren (optional)
- Erstellten Bucket öffnen
- "Berechtigung" öffnen
- "Zugriffsrechte erteilen" klicken
- In dem Eingabefeld "Neue Hauptkonten" die E-Mail des Service Accounts aus dem anderen Projekt einfügen
- Im Feld "Rolle auswählen" die Rolle "Storage Object User" auswählen

