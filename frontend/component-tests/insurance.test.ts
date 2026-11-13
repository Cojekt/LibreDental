import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup, within } from "@testing-library/svelte";
import InsuranceSection from "../src/views/billing/InsuranceFeeOverridesSection.svelte";
import PricingPicker from "../src/components/PricingPicker.svelte";
import ChartingView from "../src/views/ChartingView.svelte";
import ClaimsSection from "../src/views/billing/ClaimsSection.svelte";
import BillingView from "../src/views/BillingView.svelte";
import FeeSchedulesSection from "../src/views/billing/FeeSchedulesSection.svelte";
import { testAuth } from "./auth.svelte";
const api = vi.hoisted(() => ({
  sessionUser: vi.fn(),
  listInsuranceProviders: vi.fn(),
  listProcedures: vi.fn(),
  providers: [] as any[],
  fees: [] as any[],
  preview: vi.fn(),
  saveProvider: vi.fn(),
  saveFee: vi.fn(),
  deleteFee: vi.fn(),
  saveTreatment: vi.fn(),
  createClaim: vi.fn(),
  updateClaim: vi.fn(),
  claims: [] as any[],
  treatments: [] as any[],
}));
vi.mock("../src/stores/auth.svelte.js", async () => ({
  auth: (await import("./auth.svelte")).testAuth,
}));
vi.mock("$lib/currency.js", () => ({ formatCurrency: (n: number) => `$${(n / 100).toFixed(2)}` }));
vi.mock("@bindings/services/index.js", () => ({
  AuditService: { GetSessionUser: api.sessionUser },
  InsuranceService: {
    ListProviders: api.listInsuranceProviders,
    ListFees: vi.fn(async () => api.fees),
    SaveProvider: api.saveProvider,
    SaveFee: api.saveFee,
    DeleteFee: api.deleteFee,
    Preview: api.preview,
  },
  BillingService: {
    CreateClaim: api.createClaim,
    UpdateClaim: api.updateClaim,
    ListProcedureCodes: api.listProcedures,
    ListFeeSchedules: vi.fn(async () => []),
    ListProviders: vi.fn(async () => []),
    ListAvailableProviders: vi.fn(async () => []),
    ListClaims: vi.fn(async () => api.claims),
  },
  ChartService: {
    GetPatientChart: vi.fn(async () => ({ conditions: api.treatments })),
    SaveToothCondition: api.saveTreatment,
  },
  DocumentService: { ListPatientDocuments: vi.fn(async () => []) },
}));
const meta = { code: "US", default_currency: "USD" } as any;
beforeEach(() => {
  vi.clearAllMocks();
  testAuth.token = "session";
  api.sessionUser.mockImplementation(async (token: string) =>
    token === "session" ? { id: "dentist" } : null
  );
  api.listInsuranceProviders.mockImplementation(async () => api.providers);
  api.listProcedures.mockImplementation(async () => [
    { code: "D2391", description: "Filling", default_fee: 15000, effective_fee: 14000 },
  ]);
  api.providers = [{ id: "insurer", name: "Example Insurance", archived: false }];
  api.treatments = [];
  api.saveTreatment.mockImplementation(async (_t: any, c: any) => {
    api.treatments.push(c);
    return c;
  });
  api.createClaim.mockImplementation(async (_t: any, c: any) => c);
  api.updateClaim.mockImplementation(async (_t: any, c: any) => c);
  api.claims = [];
  api.fees = [];
  api.preview.mockImplementation(async (_token: any, p: any) => ({
    ...p,
    provider_name: p.kind === "insurance" ? "Example Insurance" : "",
    has_base: true,
    base_fee: 15000,
    standard_fee: 14000,
    amount: p.kind === "insurance" ? 10000 : p.kind === "standard" ? 14000 : p.amount,
    version: "current",
    confirmed_at: "0001-01-01T00:00:00Z",
  }));
  api.saveProvider.mockImplementation(async (_t: any, p: any) => {
    const saved = { ...p, id: p.id || "new-insurer" };
    api.providers = [...api.providers.filter((x) => x.id !== saved.id), saved];
    return saved;
  });
  api.saveFee.mockImplementation(async (_t: any, f: any) => {
    api.fees = [f];
  });
  api.deleteFee.mockImplementation(async () => {
    api.fees = [];
  });
});
afterEach(cleanup);
it("displays a zero-dollar standard override without substituting the base price", async () => {
  api.listProcedures.mockResolvedValue([
    { code: "D2391", description: "Filling", default_fee: 15000, effective_fee: 0 },
  ]);
  render(FeeSchedulesSection, { providers: [], countryMeta: meta });
  const row = (await screen.findByText("D2391")).closest("tr")!;
  expect(within(row).getByText("$0.00")).toBeTruthy();
  expect(within(row).getByText("$150.00")).toBeTruthy();
  expect(within(row).getByText("Custom")).toBeTruthy();
});
it("opens the insurance tab beside Fee Schedules", async () => {
  render(BillingView, { patients: [], providers: [], countryMeta: meta });
  await fireEvent.click(screen.getByRole("button", { name: "Insurance Fees" }));
  await screen.findByText("Example Insurance");
});
it("searches by provider name and creates a missing provider", async () => {
  render(InsuranceSection, { countryMeta: meta });
  await screen.findByText("Example Insurance");
  await fireEvent.input(screen.getByRole("textbox", { name: "Search insurance providers" }), {
    target: { value: "example" },
  });
  expect(screen.getByText("Example Insurance")).toBeTruthy();
  await fireEvent.input(screen.getByRole("textbox", { name: "Insurance provider name" }), {
    target: { value: "Another Insurer" },
  });
  await waitFor(() =>
    expect(
      (screen.getByRole("button", { name: "Add provider" }) as HTMLButtonElement).disabled
    ).toBe(false)
  );
  await fireEvent.click(screen.getByRole("button", { name: "Add provider" }));
  await waitFor(() =>
    expect(api.saveProvider).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({ name: "Another Insurer" })
    )
  );
});
it("renames and archives/restores an insurer without creating duplicates", async () => {
  render(InsuranceSection, { countryMeta: meta });
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.click(screen.getByRole("button", { name: "Rename" }));
  await fireEvent.input(screen.getByRole("textbox", { name: "Insurance provider name" }), {
    target: { value: "Renamed" },
  });
  await fireEvent.click(
    within(
      screen.getByRole("textbox", { name: "Insurance provider name" }).closest("form")!
    ).getByRole("button", { name: "Save" })
  );
  await waitFor(() => expect(api.providers[0].name).toBe("Renamed"));
  await fireEvent.click(screen.getByRole("button", { name: "Archive" }));
  await waitFor(() => expect(api.providers[0].archived).toBe(true));
  await fireEvent.click(screen.getByRole("checkbox", { name: "Show archived entries" }));
  await fireEvent.click(screen.getByRole("button", { name: "Restore" }));
  await waitFor(() => expect(api.providers[0].archived).toBe(false));
  expect(api.providers).toHaveLength(1);
});
it("distinguishes missing from zero, saves zero and removes only the override", async () => {
  render(InsuranceSection, { countryMeta: meta });
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await screen.findByPlaceholderText("No configured fee");
  expect(screen.getByText("$150.00")).toBeTruthy();
  await fireEvent.input(screen.getByRole("spinbutton", { name: "Fee for D2391" }), {
    target: { value: "0" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save" }));
  await waitFor(() =>
    expect(api.saveFee).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({ fee: 0, provider_id: "insurer" })
    )
  );
  await screen.findByRole("button", { name: "Remove override" });
  expect(screen.queryByPlaceholderText("No configured fee")).toBeNull();
  await fireEvent.click(screen.getByRole("button", { name: "Remove override" }));
  await waitFor(() =>
    expect(api.deleteFee).toHaveBeenCalledWith("session", "insurer", "US", "D2391")
  );
});
it("shows duplicate/provider errors", async () => {
  api.saveProvider.mockRejectedValue(new Error("Duplicate name"));
  render(InsuranceSection, { countryMeta: meta });
  await screen.findByText("Example Insurance");
  await fireEvent.input(screen.getByRole("textbox", { name: "Insurance provider name" }), {
    target: { value: "Example Insurance" },
  });
  await waitFor(() =>
    expect(
      (screen.getByRole("button", { name: "Add provider" }) as HTMLButtonElement).disabled
    ).toBe(false)
  );
  await fireEvent.click(screen.getByRole("button", { name: "Add provider" }));
  expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Duplicate name");
});
function picker(onconfirm = vi.fn(async () => {})) {
  return {
    onconfirm,
    props: {
      showModal: true,
      patientID: "patient-a",
      patientName: "Alice Example",
      code: "D2391",
      description: "Filling",
      countryMeta: meta,
      manualFee: 14000,
      onconfirm,
    },
  };
}
it("shows original and effective standard prices; cancel never applies a charge", async () => {
  const p = picker();
  render(PricingPicker, p.props);
  await screen.findByText("$150.00");
  expect(screen.getByText(/Alice Example/)).toBeTruthy();
  expect(screen.queryByText("Latest Used")).toBeNull();
  await screen.findByRole("button", { name: "Confirm" });
  await fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(p.onconfirm).not.toHaveBeenCalled();
});
it("confirms current standard pricing exactly once while saving", async () => {
  let finish!: () => void;
  const onconfirm = vi.fn(
    () =>
      new Promise<void>((r) => {
        finish = r;
      })
  );
  const p = picker(onconfirm);
  render(PricingPicker, p.props);
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  const button = screen.getByRole("button", { name: "Confirm" });
  await fireEvent.click(button);
  expect(onconfirm).toHaveBeenCalledWith(
    expect.objectContaining({ kind: "standard", amount: 14000 })
  );
  expect((button as HTMLButtonElement).disabled).toBe(true);
  finish();
});
it("does not silently retry stale confirmation failures", async () => {
  const onconfirm = vi.fn(async () => {
    throw new Error("Price changed; confirm again");
  });
  render(PricingPicker, picker(onconfirm).props);
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    "Price changed; confirm again"
  );
  expect(onconfirm).toHaveBeenCalledTimes(1);
  expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
    true
  );
});

const patient = {
  id: "patient-a",
  first_name: "Alice",
  last_name: "Example",
  date_of_birth: "1990-01-01T00:00:00Z",
} as any;
it("saves a real treatment-form selection only after insurance price confirmation", async () => {
  const view = render(ChartingView, { patients: [patient], countryMeta: meta });
  await fireEvent.change(screen.getByLabelText("Patient:"), { target: { value: "patient-a" } });
  await fireEvent.click(await screen.findByRole("button", { name: /^3(?:\s|$)/ }));
  await fireEvent.change(view.container.querySelector("#condition-catalog-code")!, {
    target: { value: "D2391" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save" }));
  await screen.findByText("$150.00");
  expect(api.saveTreatment).not.toHaveBeenCalled();
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.click(screen.getByRole("button", { name: "Review selected insurance price" }));
  await screen.findByText("$100.00");
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() =>
    expect(api.saveTreatment).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        patient_id: "patient-a",
        ada_code: "D2391",
        fee: 10000,
        pricing: expect.objectContaining({ kind: "insurance", provider_id: "insurer" }),
      })
    )
  );
});
it("confirms and saves a direct claim with insurance pricing", async () => {
  render(ClaimsSection, { patients: [patient], providers: [], countryMeta: meta });
  await fireEvent.click(screen.getByRole("button", { name: "New Claim" }));
  await fireEvent.click(screen.getByRole("button", { name: "+ Add Row" }));
  await fireEvent.input(screen.getByRole("textbox", { name: "Procedure Code" }), {
    target: { value: "D2391" },
  });
  await fireEvent.input(screen.getByRole("textbox", { name: "Description" }), {
    target: { value: "Filling" },
  });
  await fireEvent.input(screen.getByRole("spinbutton", { name: "Fee" }), {
    target: { value: "140" },
  });
  const dialog = screen.getByRole("dialog");
  await fireEvent.click(within(dialog).getByRole("button", { name: "New Claim" }));
  await screen.findByText("$150.00");
  expect(api.createClaim).not.toHaveBeenCalled();
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.click(screen.getByRole("button", { name: "Review selected insurance price" }));
  await screen.findByText("$100.00");
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() =>
    expect(api.createClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        patient_id: "patient-a",
        line_items: [
          expect.objectContaining({
            ada_code: "D2391",
            fee: 10000,
            pricing: expect.objectContaining({ provider_id: "insurer" }),
          }),
        ],
      })
    )
  );
});
async function editExistingClaim(pricing?: any, code = "D2391") {
  api.claims = [
    {
      id: "historical-claim",
      patient_id: patient.id,
      provider_id: "",
      date_of_service: "2026-01-01",
      status: "draft",
      notes: "Original note",
      created_at: "2026-01-01T00:00:00Z",
      line_items: [
        { id: "old-line", ada_code: code, description: "Historical filling", fee: 12345, pricing },
      ],
    },
  ];
  render(ClaimsSection, { patients: [patient], providers: [], countryMeta: meta });
  await fireEvent.click(await screen.findByRole("button", { name: "Edit", exact: true }));
  await screen.findByRole("heading", { name: "Edit Claim" });
}
it("saves legacy claim notes without price review or changing the historical amount", async () => {
  await editExistingClaim();
  await fireEvent.input(screen.getByRole("textbox", { name: "Clinical & Administrative Notes" }), {
    target: { value: "Updated note" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        notes: "Updated note",
        line_items: [expect.objectContaining({ id: "old-line", fee: 12345, pricing: undefined })],
      })
    )
  );
  expect(api.preview).not.toHaveBeenCalled();
});
it("preserves legacy prices when non-pricing insurance and status fields change", async () => {
  await editExistingClaim();
  await fireEvent.input(document.querySelector("#cl-carrier")!, {
    target: { value: "Updated carrier text" },
  });
  await fireEvent.change(document.querySelector("#cl-status")!, { target: { value: "accepted" } });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        insurance_carrier: "Updated carrier text",
        status: "accepted",
        line_items: [expect.objectContaining({ fee: 12345, pricing: undefined })],
      })
    )
  );
  expect(api.preview).not.toHaveBeenCalled();
});
it("preserves an unchanged legacy manual line with no procedure code", async () => {
  await editExistingClaim(undefined, "");
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        line_items: [expect.objectContaining({ ada_code: "", fee: 12345, pricing: undefined })],
      })
    )
  );
  expect(api.preview).not.toHaveBeenCalled();
});
it("requires confirmation when the procedure on a legacy line changes", async () => {
  await editExistingClaim();
  await fireEvent.input(screen.getByRole("textbox", { name: "Procedure Code" }), {
    target: { value: "D0120" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await screen.findByRole("heading", { name: "Review and confirm procedure price" });
  expect(api.updateClaim).not.toHaveBeenCalled();
  expect(api.preview).toHaveBeenCalledWith("session", expect.objectContaining({ code: "D0120" }));
});
it("preserves an unchanged insured claim's historical source after current fees change", async () => {
  const pricing = {
    kind: "insurance",
    patient_id: patient.id,
    clinician_id: "",
    code: "D2391",
    amount: 12345,
    provider_id: "insurer",
    provider_name: "Historical insurer",
    version: "old",
    confirmed_at: "2026-01-01T00:00:00Z",
  };
  await editExistingClaim(pricing);
  await fireEvent.input(screen.getByRole("textbox", { name: "Clinical & Administrative Notes" }), {
    target: { value: "Another note" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        line_items: [expect.objectContaining({ fee: 12345, pricing })],
      })
    )
  );
  expect(api.preview).not.toHaveBeenCalled();
});
it("requires confirmation for a deliberate price change to a legacy claim line", async () => {
  await editExistingClaim();
  await fireEvent.input(screen.getByRole("spinbutton", { name: "Fee" }), {
    target: { value: "222" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await screen.findByRole("heading", { name: "Review and confirm procedure price" });
  expect(api.updateClaim).not.toHaveBeenCalled();
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        line_items: [
          expect.objectContaining({
            fee: 22200,
            pricing: expect.objectContaining({ kind: "manual", amount: 22200 }),
          }),
        ],
      })
    )
  );
});
it("confirms only a newly added claim line and preserves the legacy line", async () => {
  await editExistingClaim();
  await fireEvent.click(screen.getByRole("button", { name: "+ Add Row" }));
  await fireEvent.input(screen.getAllByRole("textbox", { name: "Procedure Code" })[1], {
    target: { value: "D2391" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await screen.findByRole("heading", { name: "Review and confirm procedure price" });
  expect(api.updateClaim).not.toHaveBeenCalled();
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() =>
    expect(api.updateClaim).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        line_items: [
          expect.objectContaining({ id: "old-line", fee: 12345, pricing: undefined }),
          expect.objectContaining({ pricing: expect.any(Object) }),
        ],
      })
    )
  );
  expect(api.preview.mock.calls.some(([, p]) => p.amount === 12345)).toBe(false);
});
it("clears a stale claim error after successful renewed confirmation", async () => {
  api.createClaim.mockRejectedValueOnce({ cause: { code: "pricing_changed" } });
  render(ClaimsSection, { patients: [patient], providers: [], countryMeta: meta });
  await fireEvent.click(screen.getByRole("button", { name: "New Claim" }));
  await fireEvent.click(screen.getByRole("button", { name: "+ Add Row" }));
  await fireEvent.input(screen.getByRole("textbox", { name: "Procedure Code" }), {
    target: { value: "D2391" },
  });
  await fireEvent.click(
    within(screen.getByRole("dialog")).getByRole("button", { name: "New Claim" })
  );
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.click(screen.getByRole("button", { name: "Review selected insurance price" }));
  await screen.findAllByText("$100.00");
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await screen.findAllByText("The price changed. Review the current amount and confirm again.");
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.click(screen.getByRole("button", { name: "Review selected insurance price" }));
  await screen.findAllByText("$100.00");
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() => expect(api.createClaim).toHaveBeenCalledTimes(2));
  await waitFor(() =>
    expect(
      screen.queryByText("The price changed. Review the current amount and confirm again.")
    ).toBeNull()
  );
});
it("replaces logged-out insurance management with the existing sign-in message", async () => {
  testAuth.token = "";
  render(BillingView, { patients: [], providers: [], countryMeta: meta });
  await fireEvent.click(screen.getByRole("button", { name: "Insurance Fees" }));
  await screen.findByText("Sign in to make changes");
  expect(screen.queryByRole("button", { name: "Add provider" })).toBeNull();
  expect(screen.queryByRole("textbox", { name: "Insurance provider name" })).toBeNull();
  expect(api.listInsuranceProviders).not.toHaveBeenCalled();
  expect(api.sessionUser).not.toHaveBeenCalled();
});
it("rejects an invalid stored session and restores management after login", async () => {
  testAuth.token = "invalid";
  render(InsuranceSection, { countryMeta: meta });
  await waitFor(() => expect(api.sessionUser).toHaveBeenCalledWith("invalid"));
  expect(screen.queryByRole("button", { name: "Add provider" })).toBeNull();
  expect(api.listInsuranceProviders).not.toHaveBeenCalled();
  testAuth.token = "session";
  await screen.findByRole("button", { name: "Example Insurance" });
  expect(screen.getByRole("button", { name: "Add provider" })).toBeTruthy();
  testAuth.token = "";
  await screen.findByText("Sign in to make changes");
  expect(screen.queryByRole("button", { name: "Add provider" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Example Insurance" })).toBeNull();
});
it.each(["", "invalid"])(
  "blocks logged-out or invalid-session treatment/claim price selection (%s)",
  async (token) => {
    testAuth.token = token;
    const confirm = vi.fn();
    render(PricingPicker, {
      showModal: true,
      patientID: "patient-a",
      patientName: "Patient A",
      code: "D2391",
      description: "Filling",
      countryMeta: meta,
      onconfirm: confirm,
    });
    await screen.findByText("Sign in to make changes");
    expect(screen.queryByRole("button", { name: "Confirm" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Standard pricing" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add provider" })).toBeNull();
    expect(api.preview).not.toHaveBeenCalled();
    expect(confirm).not.toHaveBeenCalled();
  }
);
it("removes a reviewed price and confirmation controls immediately on logout", async () => {
  const confirm = vi.fn();
  render(PricingPicker, {
    showModal: true,
    patientID: "patient-a",
    patientName: "Patient A",
    code: "D2391",
    description: "Filling",
    countryMeta: meta,
    manualFee: 14000,
    onconfirm: confirm,
  });
  await screen.findByRole("button", { name: "Example Insurance" });
  await fireEvent.click(screen.getByRole("button", { name: "Review selected insurance price" }));
  testAuth.token = "";
  await screen.findByText("Sign in to make changes");
  expect(screen.queryByRole("button", { name: "Confirm" })).toBeNull();
  expect(screen.queryByText("Patient A")).toBeNull();
  expect(confirm).not.toHaveBeenCalled();
});
it("rejects fractional cents rather than silently rounding an insurance fee", async () => {
  render(InsuranceSection, { countryMeta: meta });
  await fireEvent.click(await screen.findByRole("button", { name: "Example Insurance" }));
  await fireEvent.input(screen.getByRole("spinbutton", { name: "Fee for D2391" }), {
    target: { value: "1.005" },
  });
  expect((screen.getByRole("button", { name: "Save" }) as HTMLButtonElement).disabled).toBe(true);
  expect(api.saveFee).not.toHaveBeenCalled();
});
it("keeps a zero-dollar uncatalogued procedure on manual pricing", async () => {
  api.preview.mockImplementation(async (_token, p) => {
    if (p.kind !== "manual") throw new Error("No base procedure");
    return {
      ...p,
      has_base: false,
      base_fee: 0,
      standard_fee: 0,
      amount: 0,
      version: "manual-zero",
    };
  });
  const confirm = vi.fn();
  render(PricingPicker, {
    showModal: true,
    patientID: patient.id,
    patientName: "Alice Example",
    code: "PROC",
    description: "Manual procedure",
    manualFee: 0,
    countryMeta: meta,
    onconfirm: confirm,
  });
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  expect(confirm).toHaveBeenCalledWith(expect.objectContaining({ kind: "manual", amount: 0 }));
});
it("makes chart-linked procedure codes and fees read-only", async () => {
  api.claims = [
    {
      id: "linked",
      patient_id: patient.id,
      provider_id: "",
      date_of_service: "2026-01-01",
      status: "draft",
      line_items: [
        {
          id: "line",
          tooth_condition_id: "condition",
          ada_code: "D2391",
          description: "Linked filling",
          fee: 12345,
        },
      ],
    },
  ];
  render(ClaimsSection, { patients: [patient], providers: [], countryMeta: meta });
  await fireEvent.click(await screen.findByRole("button", { name: "Edit", exact: true }));
  expect(
    (screen.getByRole("textbox", { name: "Procedure Code" }) as HTMLInputElement).disabled
  ).toBe(true);
  expect((screen.getByRole("spinbutton", { name: "Fee" }) as HTMLInputElement).disabled).toBe(true);
});
it("shows failed notes-only saves inside the claim modal without an unhandled rejection", async () => {
  await editExistingClaim();
  api.updateClaim.mockRejectedValueOnce({});
  await fireEvent.click(screen.getByRole("button", { name: "Save Changes" }));
  await within(screen.getByRole("dialog")).findByText(
    "Unable to complete the request. Review the details and try again."
  );
});
it("rejects an empty chart fee and clears its error when reopening the form", async () => {
  const view = render(ChartingView, { patients: [patient], countryMeta: meta });
  await fireEvent.change(screen.getByLabelText("Patient:"), { target: { value: patient.id } });
  await fireEvent.click(await screen.findByRole("button", { name: /^3(?:\s|$)/ }));
  await fireEvent.input(view.container.querySelector("#condition-description")!, {
    target: { value: "Synthetic validation finding" },
  });
  await fireEvent.input(view.container.querySelector("#condition-fee")!, { target: { value: "" } });
  await fireEvent.click(screen.getByRole("button", { name: "Save", exact: true }));
  await screen.findByText("Enter a valid nonnegative fee with no more than two decimal places.");
  expect(api.saveTreatment).not.toHaveBeenCalled();
  expect(api.preview).not.toHaveBeenCalled();
  await fireEvent.click(screen.getByRole("button", { name: "Cancel", exact: true }));
  await fireEvent.click(screen.getByRole("button", { name: /^3(?:\s|$)/ }));
  expect(
    screen.queryByText("Enter a valid nonnegative fee with no more than two decimal places.")
  ).toBeNull();
});
it("accepts an intentionally entered zero chart fee after explicit confirmation", async () => {
  const view = render(ChartingView, { patients: [patient], countryMeta: meta });
  await fireEvent.change(screen.getByLabelText("Patient:"), { target: { value: patient.id } });
  await fireEvent.click(await screen.findByRole("button", { name: /^3(?:\s|$)/ }));
  await fireEvent.change(view.container.querySelector("#condition-catalog-code")!, {
    target: { value: "D2391" },
  });
  await fireEvent.input(view.container.querySelector("#condition-fee")!, {
    target: { value: "0" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save", exact: true }));
  await waitFor(() =>
    expect((screen.getByRole("button", { name: "Confirm" }) as HTMLButtonElement).disabled).toBe(
      false
    )
  );
  expect(api.saveTreatment).not.toHaveBeenCalled();
  await fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
  await waitFor(() =>
    expect(api.saveTreatment).toHaveBeenCalledWith(
      "session",
      expect.objectContaining({
        fee: 0,
        pricing: expect.objectContaining({ kind: "manual", amount: 0 }),
      })
    )
  );
});
