let auth0 = null;
let accessToken = '';

window.onload = async () => {
  const ui = SwaggerUIBundle({
    "dom_id": "#swagger-ui",
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    validatorUrl: null,
    url: "docs/openapi.yml",
  })
  window.ui = ui

  await configureClient();

  updateUI();
}

const configureClient = async () => {
  const response = await fetchAuthConfig();
  const config = await response.json();
  // createAuth0Client comes from CDN
  auth0 = await createAuth0Client({
    domain: config.domain,
    client_id: config.clientId,
    audience: config.audience,
  });
}

const fetchAuthConfig = () => fetch("/auth_config.json");

/**
 * show or hide buttons based on authentication status
 */
const updateUI = async () => {
  const isAuthenticated = await auth0.isAuthenticated();

  document.getElementById("btn-logout").disabled = !isAuthenticated;
  document.getElementById("btn-login").disabled = isAuthenticated;
  document.getElementById("btn-copy").disabled = !isAuthenticated;
  document.getElementById('btn-copy').hidden = !isAuthenticated;
};

const login = async () => {
  await auth0.loginWithPopup();
  const isAuthenticated = await auth0.isAuthenticated();
  if (isAuthenticated) {
    try {
      accessToken = await auth0.getTokenSilently({
        audience: 'https://qgraph.org/api',
      });
    } catch (err) {
      console.log(err);
    }
    if (!accessToken) {
      window.alert('Failed to get user access token.');
    } else {
      console.log(accessToken);
    }
  }
  updateUI();
};

const logout = () => {
  auth0.logout({
    returnTo: window.location.origin
  });
  accessToken = '';
};

/**
 * Copy user access token to clipboard for easy pasting
 * into api docs
 */
async function copyToClipboard() {
  const textarea = document.createElement('textarea');
  textarea.innerHTML = accessToken;
  document.body.appendChild(textarea);
  textarea.select();
  // focus is needed in case copying is done from modal
  // also needs to come after select for unknown reason
  textarea.focus();
  document.execCommand('copy');
  textarea.remove();
}
