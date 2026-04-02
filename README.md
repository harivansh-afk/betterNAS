betterNAS is a hosted control plane with a user-run node agent.
The control plane owns user auth, node enrollment, heartbeats, export state, and mount issuance.
The node agent runs on the machine that owns the files and serves them over WebDAV.
The web app reads from the control plane and shows nodes, exports, and mount details.
Finder mounts the export from the node's public WebDAV URL using the same betterNAS username and password.
File traffic goes directly between the client and the node, not through the control plane.

<img width="1330" height="614" alt="image" src="https://github.com/user-attachments/assets/f4cfe135-505f-4ce1-bbb3-1f1d47821f8f" />
