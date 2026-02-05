Notes to get this running:
- Auth0 test users need to have Unassigned New User role assigned, otherwise they get marked as a general user and this will change their permissions and fail tests
- Running this locally - check that docker works as a normal user, on Ubuntu this involved running some docker as non-root user script, see: https://docs.docker.com/engine/security/rootless/
- Need to be sure seed data makes it to S3
