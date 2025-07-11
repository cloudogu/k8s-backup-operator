# Set up a backup bucket incl. access in the google cloud

## Setting up with Terraform

An example of setting up a Google Cloud bucket with Terraform can be found at
[here](https://github.com/cloudogu/k8s-ecosystem/blob/develop/terraform/examples/ces_google_gke/google_bucket/README.md)


## Setting up with Google Cloud user interface

### Prerequisites
- The Cloud Key Management API must be activated in the project if encryption is required
- A service account must be available
  - Must have the role "cloudkms.cryptoKeyEncrypterDecrypter" if encryption is required
  - Key for encryption, if desired

#### Activate Cloud Key Management Service API
- Call up the KMS area of the project
- Activate Cloud Key Management Service API

#### Create service account
- Call up the IAM area of the project
- Create service account / service account
- Create name
- Set description
- Create
- Assign role "cloudkms.cryptoKeyEncrypterDecrypter" (optional, required for encryption)

#### Create key for encryption (optional)
- Switch to Security -> Key management -> Key rings
- Create a new key ring if necessary
  - Region: europe-west3
- Create key
  - Assign name
  - Generated Key
  - Symmetric encrypt/decrypt
  - Key rotation: Never (or establish a better procedure)
  - Labels
    - purpose: (what this key is to be used for)
    - key-name: $NAME (as above)
    - team: $TEAM (e.g. ces)


### Creating the bucket
- Call up buckets in the cloud storage area of the project
- Click "Create bucket"
- Assign a name
- Assign labels
  - team: $TEAM (e.g. ces)
  - purpose: $PURPOSE (what this bucket is to be used for)
  - bucket_name: $NAME (as above)
- Location type: Region europe-west3
- Storage class: Standard
  - Or nearline/coldline if there is infrequent access
- Prevent public access: Fine-grained
- Data protection: **No** soft-delete!
- Data encryption: Cloud KMS key
  - Select the key created above

### Configure access from another project (optional)
- Open the created bucket
- Open "Authorisation"
- Click on "Grant access rights"
- Insert the email of the service account from the other project in the "New main accounts" input field
- Select the "Storage Object User" role in the "Select role" field