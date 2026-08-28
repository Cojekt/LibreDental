import type { Provider } from "@bindings/domain/models.js";
import { PracticeConfigService } from "@bindings/services/index.js";

function createAuthStore() {
  let currentStaffId = $state<string | null>(null);
  let currentStaff = $state<Provider | null>(null);

  // Load initial state from sessionStorage if in browser
  if (typeof window !== "undefined") {
    const stored = sessionStorage.getItem("currentStaffId");
    if (stored) {
      currentStaffId = stored;
      // Fetch the full provider object
      fetchStaffDetails(stored);
    }
  }

  async function fetchStaffDetails(id: string) {
    try {
      const providers = await PracticeConfigService.ListProviders();
      if (currentStaffId !== id) return;
      const provider = (providers as Provider[])?.find((p) => p.id === id && p.is_active);
      if (provider) {
        currentStaff = provider;
      } else {
        // Provider might have been deleted
        logout();
      }
    } catch (e) {
      console.error("Failed to fetch staff details:", e);
      // Transient failures shouldn't log the user out; preserve the session.
    }
  }

  function login(provider: Provider) {
    currentStaffId = provider.id;
    currentStaff = provider;
    if (typeof window !== "undefined") {
      sessionStorage.setItem("currentStaffId", provider.id);
    }
  }

  function logout() {
    currentStaffId = null;
    currentStaff = null;
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("currentStaffId");
    }
  }

  return {
    get currentStaffId() {
      return currentStaffId;
    },
    get currentStaff() {
      return currentStaff;
    },
    login,
    logout,
  };
}

export const auth = createAuthStore();
