# Laforge

> Security Competition Infrastructure Automation Framework

## ![Laforge Image](https://i.etsystatic.com/6782226/r/il/89b846/1006866306/il_570xN.1006866306_680f.jpg)

Laforge enables rapid development of infrastructure for the purpose of information security competitions. Using a simple and intuitive configuration language, Laforge manages a dependency graph and state management and allows for highly productive remote collaboration. The Laforge engine uses a custom loader to do multi-dimensional, non-destructive configuration overlay. A good analogy to this is Docker - when you build a Docker container, it builds it up layers at a time. It's this power that has inspired us to build Laforge. It's certainly a niche project, but we certainly have found an incredible use for it.


## Table of Contents

- [Features](#features)
- [FAQ](#faq)
- [Acknowledgements](#acknowledgements)

## Features

- Cross platform
- Portable - installs as a stand alone native executable.
- Use what you enjoy - Bring Your Own Scripting Language (Y)
- Fast.
- Build once, clone to _n_ number of teams (security competitions paradigm)
- Collaborative - makes working in distributed groups very efficient

## FAQ

### What is Laforge?

Laforge is a framework that lets you design and implement security competitions in a scalable, collaborative, and fun way! You write configurations in _Laforge Config Language_ and use the  tool to inspect, validate, build, and connect to remote infrastructure with. Historically, Terraform was used as the "backend" (generates sophisticated terraform configurations), but this has changed over the last two years with LaForge now directly working with cloud and virtualization services to build hosts, deploy an agent, and configure those hosts in a consistent manner. Laforge currently powers all of the infrastructure management for the Collegiate Penetration Testing Competition and has supported game deployments of >4000 unique nodes.

### Why was it created?

Three reasons:

- Security professionals aren't the most well versed with operations/infrastructure/devops tools. They have a steeper than most learning curve, especially when asking volunteers to try and figure it out in their off work time. To make it easier for people, we wanted to make a tool that basically did the hard part for them.
- As we dug in, we noticed that the commonly used automation frameworks available had a number of painpoints when it came to building security competition infrastructure. There are things that have to occur in security competitions that aren't supported in the real world:
  - wide compatibility with lots of operating systems and software
  - Mass "clone" ability - snapshot a game infra and clone it 10-20x - one for each team, including networks and not just flag boxes.
  - Flexibility to deploy the same stacks to a wide set of possible infrastructure - VMWare, AWS, GCP, etc.
- Because competitions deserve it! We work with some of the most passionate people on these projects and anything that can make our shared experience better is a win win in our book.

### Why not current DevOps tools?

No need to go into a flame war over this tool or that. We frankly like them. Our biggest complaint across the board is that given how fragmented they are, it's hard to ever be really _good_ at any one of them. We needed something that volunteers could pick up quickly (like shell scripts) without having to learn a new toolset.  We also needed something that could build networks, hosts, and configurations consistently across multiple teams.  

### How does it scale?

We have used the various iterations of LaForge to generate competition environments with thousands of total hosts.  In short, it can scale as large as your imagination (and budget / resources) allows.  Furthermore, we have used this tool across teams of volunteer developers each working on their own components and have used that feedback in the most recent versions.  

### What about performance?

Depending on the complexity of your environment, building LaForge output may take seconds or minutes.  In the end you will spend more time spinning up systems in the environment of your choice with Terraform or Vagrant than you will generating the relevant configurations for either of them.  

### Is it production-ready?

If by production, you mean developing live competition environments, LaForge has been used for over six years in a "production" capacity.  If you mean live systems at your company or organization, it will probably work well, but use at your own risk.

## Hall of Fame

This project was dreamed, birthed, and originally developed by Alex Levinson (@gen0cide) for the Collegiate Penetration Testing Competition.

#### mentors, contributors, and great friends of Laforge

- @1njecti0n
- @emperorcow
- @vyrus001
- @bstax
- @cmbits
- @tomk
- @brianc
- @rossja
- @kos
- @dcam
- @davehughes
- @mbm
- @maus
- @javuto

## Acknowledgements

- [Gen0cide](https://github.com/gen0cide) who created the very first version of LaForge.
- [CPTC](https://cp.tc) and the CPTC Advisory Board who's been so patient with us over the years as we developed this tool.
- [Rochester Institute of Technology](https://www.rit.edu) For giving us a place to expiriment and advance both the technology as well as the workforce of our industry.
