#!/usr/bin/env python

# localtrusted.py - Clone a repository locally and run .gopherci/pipeline.sh

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

from yaml import load

# Pop off the executable path
sys.argv.pop(0)

repository = sys.argv.pop(0)
branch = sys.argv.pop(0)

repofile = open("%s/conf/repositories.yaml" % os.getenv("GOPHER_CONFIGDIR"))
yamldata = repofile.read()

repodata = load(yamldata)

if repository in repodata:
    repoconf = repodata[repository]

if "clone_url" not in repoconf:
    bot.Say("No 'clone_url' specified for '%s' in repositories.yaml" % repository)
    exit()
clone_url = repoconf["clone_url"]

if "keep_history" not in repoconf:
    keep_history = 7
else:
    keep_history = repoconf["keep_history"]

bot.ExtendNamespace(repository, keep_history)
bot.AddTask("git-sync", [ clone_url, branch, repository, "true" ])
bot.AddTask("localexec", [ ".gopherci/pipeline.sh" ])