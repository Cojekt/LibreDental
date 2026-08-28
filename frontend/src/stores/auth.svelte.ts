import type { Provider } from "@bindings/domain/models.js";
import { PracticeConfigService, AuditService } from "@bindings/services/index.js";

function createAuthStore() {
  let currentStaffId = $state<string | null>(null);
  let currentStaff = $state<Provider | null>(null);
  let sessionToken = $state<string>("");

  // Load initial state from localStorage if in browser
  if (typeof window !== "undefined") {
    const storedId = localStorage.getItem("currentStaffId");
    const storedToken = localStorage.getItem("sessionToken");
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
        currentStaff = provider;
        const fetchedUser = await AuditService.GetSessionUser(sessionToken);
        if (!fetchedUser) {
          sessionToken = await AuditService.CreateSession(provider);
          localStorage.setItem("sessionToken", sessionToken);
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

  async function login(provider: Provider) {
    currentStaffId = provider.id;
    currentStaff = provider;
    sessionToken = await AuditService.CreateSession(provider);
    if (typeof window !== "undefined") {
      localStorage.setItem("currentStaffId", provider.id);
      localStorage.setItem("sessionToken", sessionToken);
    }
  }

  async function logout() {
    if (sessionToken) {
      await AuditService.DestroySession(sessionToken).catch(console.error);
    }
    currentStaffId = null;
    currentStaff = null;
    sessionToken = "";
    if (typeof window !== "undefined") {
      localStorage.removeItem("currentStaffId");
      localStorage.removeItem("sessionToken");
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
