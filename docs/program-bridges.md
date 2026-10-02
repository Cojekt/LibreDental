# Program Bridges

A program bridge is a local-only integration: LibreDental starts another program installed on
the same workstation (a DICOM viewer, sensor capture software, a perio charting tool) and hands
it the current patient, and optionally some of that patient's documents. Unlike the claim and
notification integrations, a bridge never talks to a remote service.

## Prior art

There is no single cross-vendor standard for this, but the existing approaches converge on the
same shape: a per-program executable path plus a way to pass patient context.

- **Open Dental "Program Links"**: the de facto model in dental PMS software. Each bridge has
  an executable path (with a per-workstation local path override), a command-line template
  using tokens such as `[PatNum]`, `[LName]`, `[FName]`, `[Birthdate_yyyyMMdd]`, and
  bridge-specific properties (e.g. "use PatientNum or ChartNum" as the patient identifier).
  Most bridges are command-line launches; some write a patient info text file instead.
- **VDDS-media** (VDDS e.V., Germany): a published dental standard where the practice system
  invokes an imaging system's module (registered with its full path in `VDDS_MMI.INI`) and
  passes patient data through an INI transfer file.
- **IHE Invoke Image Display (IID)**: the radiology standard for "show this patient's images":
  an HTTP GET to the viewer, e.g.
  `IHEInvokeImageDisplay?requestType=PATIENT&patientID=...` or `requestType=STUDY&studyUID=...`.
  Relevant once LibreDental has a PACS or web viewer to talk to.
- **DICOM viewer command lines and URL schemes**:
  - Weasis: `Weasis '$dicom:get -l "<dir>"'`, or the `weasis://` URL scheme with the same command.
  - RadiAnt: `RadiAntViewer.exe -d "<dir>"` (folders) or `-f "<file>"` (files).
  - MicroDicom: `mDicom.exe -fd "<path>"`, or the `microdicom://` URL scheme.
  - OsiriX / Horos (macOS): `osirix://` / `horos://` URL scheme with `methodName=displayStudy`.

## Design

```
domain.ProgramBridge           interface: Name, Capabilities, DefaultConfig, BuildLaunch
services.CommandLineBridge     generic implementation: executable + argument template
services.DefaultProgramBridges presets: custom, weasis, radiant, microdicom
services.BridgeService         Wails service: ListBridges, SetBridgeConfig, LaunchBridge
storage.ProgramBridgeRepository  config persistence (program_bridges table)
```

`ProgramBridge` mirrors `ClaimProvider` and `NotificationProvider`: bridges are registered by
name, and `BuildLaunch` only *describes* how to start the program. `BridgeService` owns the side
effects (exporting documents, auditing, starting the process), so new bridges cannot skip them.

### Configuration

Config is stored in the `program_bridges` table of the main database (one row per configured
bridge: `name`, `enabled`, `path`, `args`). A bridge with no row uses its preset defaults and is
disabled. It is not kept in the keychain like clinic integrations, since none of it is secret.
Every config change is audited with the new enabled state, path and arguments.

Bridges currently only run in desktop builds, where the database lives on the same machine as
the program being launched. If bridges are ever enabled for LAN server setups, executable paths
will need a per-workstation override (as Open Dental has), since installs differ between machines.

### Argument templates

| Token          | Value                                                     |
| -------------- | --------------------------------------------------------- |
| `{patient_id}` | LibreDental patient ID                                    |
| `{first_name}` | Patient first name                                        |
| `{last_name}`  | Patient last name                                         |
| `{birth_date}` | Date of birth as `YYYYMMDD` (DICOM DA format)             |
| `{sex}`        | `M` / `F` / `O`, empty if undisclosed (DICOM codes)        |
| `{dir}`        | Directory holding the exported documents                  |
| `{files}`      | One argument per exported document (must stand alone)     |

Single or double quotes group text into one argument; backslashes are literal so Windows paths
can be pasted as-is. Quotes of the other kind are kept, which is how the Weasis preset passes
its own quoted command string.

### Safety

- The template is split into arguments **before** tokens are substituted, and the program is
  started without a shell, so patient data cannot add arguments or commands.
- `.bat` / `.cmd` files are rejected: `cmd.exe` re-parses their command line with different
  quoting rules, which would undo that isolation.
- Patient values containing control characters or `"`, or starting with `-` or `/` (option
  switches), are rejected rather than silently altered.
- Requested documents must belong to the patient. They are copied to a private
  `libredental_bridge_*` temp directory with sanitized names; copies older than 24 hours are
  removed on the next launch.
- Every launch is recorded in the audit log as an `EXPORT` on the patient (resource
  `program_bridge`) **before** the program starts; if the audit entry cannot be written, the
  program is not started. Config changes are audited as `UPDATE`s on `program_bridge_config`.
- In server builds, bridges are unavailable: the program would start on the server, not on
  the workstation the user is at.

## Not yet done

- Frontend: a settings panel for bridge config and launch buttons (patient panel, document
  viewer). The service is bound; the UI is not built yet.
- URL-scheme / HTTP launches (`weasis://`, IHE IID), which need a second launch kind.
- Transfer-file bridges (VDDS-media INI, Open Dental-style patient info files).
- Writing results back (e.g. importing images captured in the bridged program).
- macOS `.app` bundles: the path currently has to point at the binary inside
  `Contents/MacOS/`.
