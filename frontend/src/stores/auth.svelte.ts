import type { Provider } from "@bindings/domain/models.js";
import { PracticeConfigService, AuditService } from "@bindings/services/index.js";

function createAuthStore() {
  let currentStaffId = $state<string | null>(null);
  let currentStaff = $state<Provider | null>(null);
  let sessionToken = $state<string>("");

  // Load initial state from sessionStorage if in browser
  if (typeof window !== "undefined") {
    const storedId = sessionStorage.getItem("currentStaffId");
    const storedToken = sessionStorage.getItem("sessionToken");
    if (storedId && storedToken) {
      currentStaffId = storedId;
      sessionToken = storedToken;
      // Fetch the full provider object
      fetchStaffDetails(storedId);
    }
  }

  async function fetchStaffDetails(id: string) {
    const attemptToken = sessionToken;
    try {
      const providers = await PracticeConfigService.ListProviders();
      if (currentStaffId !== id || sessionToken !== attemptToken) return;
      const provider = (providers as Provider[])?.find((p) => p.id === id && p.is_active);
      if (provider) {
        const fetchedUser = await AuditService.GetSessionUser(sessionToken);
        if (currentStaffId !== id || sessionToken !== attemptToken) return;
        if (fetchedUser && fetchedUser.id === provider.id) {
          currentStaff = provider;
        } else {
          logout();
        }
      } else {
        // Provider might have been deleted
        logout();
      }
    } catch (e) {
      console.error("Failed to fetch staff details:", e);
      // Transient failures shouldn't log the user out; preserve the session.
    }
  }

  async function login(provider: Provider, pin: string) {
    const token = await AuditService.CreateSession(provider.id, pin);
    commitSession(provider, token);
  }

  function commitSession(provider: Provider, token: string) {
    const oldToken = sessionToken;
    currentStaffId = provider.id;
    currentStaff = provider;
    sessionToken = token;
    if (typeof window !== "undefined") {
      sessionStorage.setItem("currentStaffId", provider.id);
      sessionStorage.setItem("sessionToken", token);
    }
    if (oldToken) {
      AuditService.DestroySession(oldToken).catch(console.error);
    }
  }

  function logout() {
    const oldToken = sessionToken;
    currentStaffId = null;
    currentStaff = null;
    sessionToken = "";
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("currentStaffId");
      sessionStorage.removeItem("sessionToken");
    }
    if (oldToken) {
      AuditService.DestroySession(oldToken).catch(console.error);
    }
  }

  return {
    get currentStaffId() {
      return currentStaffId;
    },
    get currentStaff() {
      return currentStaff;
    },
    get token() {
      return sessionToken;
    },
    login,
    commitSession,
    logout,
  };
}

export const auth = createAuthStore();
