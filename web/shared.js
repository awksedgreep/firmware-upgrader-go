// Firmware Upgrader - Shared JavaScript Utilities

/**
 * Initialize header navigation - set active link based on current page
 */
function initializeHeader() {
  const currentPath = window.location.pathname;
  const navLinks = document.querySelectorAll(".header-nav a");

  navLinks.forEach((link) => {
    const href = link.getAttribute("href");
    // Remove query parameters for comparison
    const linkPath = href.split("?")[0];

    if (currentPath === linkPath || currentPath.endsWith(linkPath)) {
      link.classList.add("active");
    } else {
      link.classList.remove("active");
    }
  });
}

/**
 * Show a message to the user
 * @param {string} message - The message text
 * @param {string} type - Message type: 'success', 'error', 'warning', 'info'
 * @param {HTMLElement} container - Optional container element (defaults to #form-message)
 * @param {number} duration - Auto-hide duration in ms (0 = no auto-hide)
 */
function showMessage(
  message,
  type = "info",
  container = null,
  duration = 5000,
) {
  const messageEl = container || document.getElementById("form-message");
  if (!messageEl) {
    console.warn("Message container not found");
    return;
  }

  messageEl.textContent = message;
  messageEl.className = `message ${type} show`;

  if (duration > 0) {
    setTimeout(() => {
      messageEl.classList.remove("show");
    }, duration);
  }
}

/**
 * Hide a message
 * @param {HTMLElement} container - Optional container element
 */
function hideMessage(container = null) {
  const messageEl = container || document.getElementById("form-message");
  if (messageEl) {
    messageEl.classList.remove("show");
  }
}

/**
 * Format a timestamp for display
 * @param {number} timestamp - Unix timestamp
 * @returns {string} Formatted date string
 */
function formatDate(timestamp) {
  const date = new Date(timestamp * 1000);
  return date.toLocaleString();
}

/**
 * Format a relative time (e.g., "2 hours ago")
 * @param {number} timestamp - Unix timestamp
 * @returns {string} Relative time string
 */
function formatRelativeTime(timestamp) {
  const now = Math.floor(Date.now() / 1000);
  const diff = now - timestamp;

  if (diff < 60) return "just now";
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;

  return formatDate(timestamp);
}

/**
 * Debounce function - useful for search/filter inputs
 * @param {Function} func - Function to debounce
 * @param {number} wait - Wait time in ms
 * @returns {Function} Debounced function
 */
function debounce(func, wait = 300) {
  let timeout;
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
}

/**
 * Make an API call with error handling
 * @param {string} endpoint - API endpoint
 * @param {Object} options - Fetch options
 * @returns {Promise} Response JSON or throws error
 */
async function apiCall(endpoint, options = {}) {
  const defaultOptions = {
    headers: {
      "Content-Type": "application/json",
    },
  };

  const mergedOptions = { ...defaultOptions, ...options };

  try {
    const response = await fetch(endpoint, mergedOptions);

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(
        errorData.error || `HTTP ${response.status}: ${response.statusText}`,
      );
    }

    // Return empty object for 204 No Content
    if (response.status === 204) {
      return {};
    }

    return await response.json();
  } catch (error) {
    console.error(`API Error (${endpoint}):`, error);
    throw error;
  }
}

/**
 * GET request helper
 * @param {string} endpoint - API endpoint
 * @returns {Promise} Response data
 */
async function apiGet(endpoint) {
  return apiCall(endpoint, { method: "GET" });
}

/**
 * POST request helper
 * @param {string} endpoint - API endpoint
 * @param {Object} data - Request body
 * @returns {Promise} Response data
 */
async function apiPost(endpoint, data) {
  return apiCall(endpoint, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

/**
 * PUT request helper
 * @param {string} endpoint - API endpoint
 * @param {Object} data - Request body
 * @returns {Promise} Response data
 */
async function apiPut(endpoint, data) {
  return apiCall(endpoint, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

/**
 * DELETE request helper
 * @param {string} endpoint - API endpoint
 * @returns {Promise} Response data
 */
async function apiDelete(endpoint) {
  return apiCall(endpoint, { method: "DELETE" });
}

/**
 * Parse URL query parameters
 * @returns {Object} Query parameters
 */
function getQueryParams() {
  const params = new URLSearchParams(window.location.search);
  const obj = {};

  for (const [key, value] of params) {
    obj[key] = value;
  }

  return obj;
}

/**
 * Get query parameter by name
 * @param {string} name - Parameter name
 * @returns {string|null} Parameter value
 */
function getQueryParam(name) {
  return getQueryParams()[name] || null;
}

/**
 * Confirm dialog helper
 * @param {string} message - Confirmation message
 * @returns {Promise<boolean>} User's choice
 */
function confirmAction(message) {
  return Promise.resolve(confirm(message));
}

/**
 * Format bytes to human-readable size
 * @param {number} bytes - Size in bytes
 * @returns {string} Formatted size
 */
function formatBytes(bytes) {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

/**
 * Create a status badge element
 * @param {string} status - Status value
 * @param {string} label - Display label
 * @returns {HTMLElement} Badge element
 */
function createStatusBadge(status, label) {
  const badge = document.createElement("span");
  badge.className = `status-badge ${status.toLowerCase()}`;
  badge.textContent = label || status;
  return badge;
}

/**
 * Validate email address
 * @param {string} email - Email address
 * @returns {boolean} Valid email
 */
function isValidEmail(email) {
  const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return regex.test(email);
}

/**
 * Validate IP address
 * @param {string} ip - IP address
 * @returns {boolean} Valid IP
 */
function isValidIP(ip) {
  const regex = /^(\d{1,3}\.){3}\d{1,3}$|^[0-9a-fA-F:]+$/;
  return regex.test(ip);
}

/**
 * Validate MAC address
 * @param {string} mac - MAC address
 * @returns {boolean} Valid MAC
 */
function isValidMAC(mac) {
  const regex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/;
  return regex.test(mac);
}

/**
 * Copy text to clipboard
 * @param {string} text - Text to copy
 * @returns {Promise<void>}
 */
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch (error) {
    console.error("Failed to copy:", error);
    return false;
  }
}

/**
 * Initialize all shared components
 */
document.addEventListener("DOMContentLoaded", () => {
  initializeHeader();
});
