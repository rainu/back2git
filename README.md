# back2git

This tool will observe files and push a copy of them into a given git repository every time the file was changed.

## Usage

The tool reads the configuration from a config yaml. So you have to provide such a config yaml:

```bash
> back2git <config_yaml>
```

This tool runs in the foreground. If you want to run in background you have to do it by your own. There is no
service/option inside this tool to do that.

## Config

| Key | Description  |
|---|---|
| repository.url | The url of the git repository where the files will be pushed in.  |
| repository.branch | The name of the remote branch to push in.  |
| repository.path | The location of the local git repository clone.  |
| repository.auth.basic | If you want to use basic auth for the given git repository.  |
| repository.auth.basic.username | The username.  |
| repository.auth.basic.password | The plain password.  |
| repository.auth.basic.passwordCommand | The command which should be executed to get the password.  |
| repository.auth.basic.passwordCommand.name | The password program path.  |
| repository.auth.basic.passwordCommand.args | The arguments for the password program.  |
| repository.auth.token | If you want to use a token as authentication for the given git repository.  |
| repository.auth.ssh | If you want to use ssh as authentication method for the given git repository.  |
| repository.auth.ssh.username | The username.  |
| repository.auth.ssh.privateKey | The location of the private key PEM file.  |
| repository.auth.ssh.pkPassword | The password for the private key.  |
| repository.auth.ssh.pkPasswordCommand | The command which should be executed to get the private key password.  |
| repository.auth.ssh.pkPasswordCommand.name | The password program path.  |
| repository.auth.ssh.pkPasswordCommand.args | The arguments for the password program.  |
| files | The files to be observed. |

### Examples

Observe the files **/etc/hosts** and **/etc/groups** and push them to the github repository **rainu/my-configs** and use
a github access token (settings > Developer Settings > Personal Access Token > Scope (all Repo) ).
```yaml
repository:
  url: https://github.com/rainu/my-configs.git
  path: /home/rainu/backup/my-configs
  auth:
    token: ghp_...
files:
  /etc/hosts:
  /etc/groups:
```

Use username/password credentials for authentication.
```yaml
repository:
  url: https://github.com/rainu/my-configs.git
  path: /home/rainu/backup/my-configs
  auth:
    basic:
      username: rainu
      password: secret
files:
  /etc/hosts:
  /etc/groups:
```

Use github repository via ssh.
```yaml
repository:
  url: github.com/rainu/my-configs.git
  path: /home/rainu/backup/my-configs
  auth:
    ssh:
      username: git
      privateKey: /home/rainu/.ssh/id_rsa
      pkPassword: secret
files:
  /etc/hosts:
  /etc/groups:
```

Call a program to get the password.
```yaml
repository:
  url: https://github.com/rainu/my-configs.git
  path: /home/rainu/backup/my-configs
  auth:
    basic:
      username: rainu
      passwordCommand: 
        name: echo
        args:
         - '-n'
         - 'secret'
files:
  /etc/hosts:
  /etc/groups:
```