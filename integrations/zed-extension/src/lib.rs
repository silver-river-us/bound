use zed_extension_api as zed;

struct BoundExtension;

impl zed::Extension for BoundExtension {
    fn new() -> Self {
        Self
    }

    fn language_server_command(
        &mut self,
        _language_server_id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
    ) -> zed::Result<zed::Command> {
        let command = worktree
            .which("bound-lsp")
            .unwrap_or_else(|| "/Users/matiasgutierrez/.local/bin/bound-lsp".to_string());
        Ok(zed::Command {
            command,
            args: vec![],
            env: vec![],
        })
    }
}

zed::register_extension!(BoundExtension);
