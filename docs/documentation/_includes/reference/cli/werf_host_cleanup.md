{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Cleanup old unused werf cache and data of all projects on host machine.

The data include:
* Lost docker containers and images from interrupted builds.
* Old service tmp dirs, which werf creates during every build, converge and other commands.
* Local cache:
  * Remote git clones cache.
  * Git worktree cache.

It is safe to run this command periodically by automated cleanup job in parallel with other werf    
commands such as build, converge and cleanup.

{{ header }} Syntax

```shell
werf host cleanup [options]
```

{{ header }} Options

```shell
      --allowed-volume-usage=80
            Set allowed percentage of docker storage volume usage which will cause garbage          
            collection of local docker images (default 80% or $WERF_ALLOWED_VOLUME_USAGE)
      --allowed-volume-usage-margin=10
            During garbage collection werf would delete images until volume usage becomes below     
            "allowed-volume-usage - allowed-volume-usage-margin" level (default 10% or              
            $WERF_ALLOWED_VOLUME_USAGE_MARGIN)
      --dev=false
            Enable development mode (default $WERF_DEV).
            The mode allows working with project files without doing redundant commits during       
            debugging and development
      --dev-mode='simple'
            Set development mode (default $WERF_DEV_MODE or simple).
            Two development modes are supported:
            - simple: for working with the worktree state of the git repository
            - strict: for working with the index state of the git repository
      --docker-config=''
            Specify docker config directory path. Default $WERF_DOCKER_CONFIG or $DOCKER_CONFIG or  
            ~/.docker (in the order of priority)
      --docker-server-storage-path=''
            Use specified path to the local docker server storage to check docker storage volume    
            usage while performing garbage collection of local docker images (detect local docker   
            server storage path by default or use $WERF_DOCKER_SERVER_STORAGE_PATH)
      --dry-run=false
            Indicate what the command would do without actually doing that (default $WERF_DRY_RUN)
      --force=false
            Force deletion of images which are being used by some containers (default $WERF_FORCE)
      --home-dir=''
            Use specified dir to store werf cache files and dirs (default $WERF_HOME or ~/.werf)
      --log-color-mode='auto'
            Set log color mode.
            Supported on, off and auto (based on the stdout’s file descriptor referring to a        
            terminal) modes.
            Default $WERF_LOG_COLOR_MODE or auto mode.
      --log-debug=false
            Enable debug (default $WERF_LOG_DEBUG).
      --log-pretty=true
            Enable emojis, auto line wrapping and log process border (default $WERF_LOG_PRETTY or   
            true).
      --log-quiet=false
            Disable explanatory output (default $WERF_LOG_QUIET).
      --log-terminal-width=-1
            Set log terminal width.
            Defaults to:
            * $WERF_LOG_TERMINAL_WIDTH
            * interactive terminal width or 140
      --log-verbose=false
            Enable verbose output (default $WERF_LOG_VERBOSE).
      --loose-giterminism=false
            Loose werf giterminism mode restrictions (NOTE: not all restrictions can be removed,    
            more info https://werf.io/documentation/advanced/giterminism.html, default              
            $WERF_LOOSE_GITERMINISM)
      --tmp-dir=''
            Use specified dir to store tmp files and dirs (default $WERF_TMP_DIR or system tmp dir)
```

