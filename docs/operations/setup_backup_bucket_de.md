# Aufsetzen eines Backup-Buckets inkl. Zugriff

## Aufsetzen mit Terraform

Zum Aufsetzen eines Google Cloud Buckets mit Terraform ist 
[hier](https://github.com/cloudogu/k8s-ecosystem/blob/develop/terraform/examples/ces_google_gke/google_bucket/README.md)
ein Beispiel zu finden.

## Aufsetzen mit Google Cloud Benutzeroberfläche

### Voraussetzungen
- Im Projekt muss die Cloud Key Management API aktiviert sein
- Ein Service Account / Dienstkonto muss vorhanden sein 
  - Muss die Rolle "cloudkms.cryptoKeyEncrypterDecrypter" besitzen, wenn Verschlüsselung erwünscht
  - Key zum Verschlüsseln, falls erwünscht

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
- Data protection: Kein soft-delete!
- Data encryption: Cloud KMS key
    - Oben angelegten Key auswählen

### Zugriff aus einem anderen Projekt konfigurieren (optional)
- Erstellten Bucket öffnen
- "Berechtigung" öffnen
- "Zugriffsrechte erteilen" klicken
- In dem Eingabefeld "Neue Hauptkonten" die E-Mail des Service Accounts aus dem anderen Projekt einfügen
- Im Feld "Rolle auswählen" die Rolle "Storage Object User" auswählen

