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
    try {
      const providers = await PracticeConfigService.ListProviders();
      if (currentStaffId !== id) return;
      const provider = (providers as Provider[])?.find((p) => p.id === id && p.is_active);
      if (provider) {
        const fetchedUser = await AuditService.GetSessionUser(sessionToken);
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
    currentStaffId = provider.id;
    currentStaff = provider;
    sessionToken = token;
    if (typeof window !== "undefined") {
      sessionStorage.setItem("currentStaffId", provider.id);
      sessionStorage.setItem("sessionToken", token);
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
    logout,
  };
}

export const auth = createAuthStore();
