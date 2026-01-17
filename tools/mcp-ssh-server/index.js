#!/usr/bin/env node

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { spawn } from "child_process";

// Configuration from environment variables
const SSH_HOST = process.env.SSH_HOST || "172.20.0.10";
const SSH_USER = process.env.SSH_USER || "basphere";
const SSH_KEY = process.env.SSH_KEY || "";
const SSH_PORT = process.env.SSH_PORT || "22";

// Create MCP server
const server = new Server(
  {
    name: "basphere-ssh",
    version: "1.0.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Execute SSH command
function executeSSH(command, useSudo = false) {
  return new Promise((resolve, reject) => {
    const actualCommand = useSudo ? `sudo ${command}` : command;

    const sshArgs = [
      "-o", "BatchMode=yes",
      "-o", "ConnectTimeout=10",
      "-o", "StrictHostKeyChecking=accept-new",
      "-p", SSH_PORT,
    ];

    if (SSH_KEY) {
      sshArgs.push("-i", SSH_KEY);
    }

    sshArgs.push(`${SSH_USER}@${SSH_HOST}`, actualCommand);

    const ssh = spawn("ssh", sshArgs);

    let stdout = "";
    let stderr = "";

    ssh.stdout.on("data", (data) => {
      stdout += data.toString();
    });

    ssh.stderr.on("data", (data) => {
      stderr += data.toString();
    });

    ssh.on("close", (code) => {
      resolve({
        exitCode: code,
        stdout: stdout.trim(),
        stderr: stderr.trim(),
      });
    });

    ssh.on("error", (err) => {
      reject(err);
    });

    // Timeout after 60 seconds
    setTimeout(() => {
      ssh.kill();
      reject(new Error("SSH command timed out after 60 seconds"));
    }, 60000);
  });
}

// List available tools
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "ssh_exec",
        description: `Execute a command on Bastion server (${SSH_USER}@${SSH_HOST})`,
        inputSchema: {
          type: "object",
          properties: {
            command: {
              type: "string",
              description: "The command to execute on the remote server",
            },
          },
          required: ["command"],
        },
      },
      {
        name: "ssh_exec_sudo",
        description: `Execute a command with sudo on Bastion server (${SSH_USER}@${SSH_HOST})`,
        inputSchema: {
          type: "object",
          properties: {
            command: {
              type: "string",
              description: "The command to execute with sudo on the remote server",
            },
          },
          required: ["command"],
        },
      },
      {
        name: "ssh_read_file",
        description: "Read a file from the Bastion server",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "The absolute path to the file",
            },
          },
          required: ["path"],
        },
      },
      {
        name: "ssh_write_file",
        description: "Write content to a file on the Bastion server (requires sudo)",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "The absolute path to the file",
            },
            content: {
              type: "string",
              description: "The content to write",
            },
          },
          required: ["path", "content"],
        },
      },
      {
        name: "ssh_list_dir",
        description: "List directory contents on the Bastion server",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "The directory path to list",
            },
          },
          required: ["path"],
        },
      },
    ],
  };
});

// Handle tool calls
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    switch (name) {
      case "ssh_exec": {
        const result = await executeSSH(args.command, false);
        return {
          content: [
            {
              type: "text",
              text: formatResult(result),
            },
          ],
        };
      }

      case "ssh_exec_sudo": {
        const result = await executeSSH(args.command, true);
        return {
          content: [
            {
              type: "text",
              text: formatResult(result),
            },
          ],
        };
      }

      case "ssh_read_file": {
        const result = await executeSSH(`cat "${args.path}"`, false);
        if (result.exitCode !== 0) {
          // Try with sudo
          const sudoResult = await executeSSH(`cat "${args.path}"`, true);
          return {
            content: [
              {
                type: "text",
                text: sudoResult.exitCode === 0 ? sudoResult.stdout : formatResult(sudoResult),
              },
            ],
          };
        }
        return {
          content: [
            {
              type: "text",
              text: result.stdout,
            },
          ],
        };
      }

      case "ssh_write_file": {
        // Use heredoc to write file content
        const escapedContent = args.content.replace(/'/g, "'\\''");
        const command = `cat > "${args.path}" << 'BASPHERE_EOF'\n${args.content}\nBASPHERE_EOF`;
        const result = await executeSSH(command, true);
        return {
          content: [
            {
              type: "text",
              text: result.exitCode === 0
                ? `File written successfully: ${args.path}`
                : formatResult(result),
            },
          ],
        };
      }

      case "ssh_list_dir": {
        const result = await executeSSH(`ls -la "${args.path}"`, false);
        return {
          content: [
            {
              type: "text",
              text: formatResult(result),
            },
          ],
        };
      }

      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  } catch (error) {
    return {
      content: [
        {
          type: "text",
          text: `Error: ${error.message}`,
        },
      ],
      isError: true,
    };
  }
});

function formatResult(result) {
  let output = "";
  if (result.stdout) {
    output += result.stdout;
  }
  if (result.stderr) {
    output += (output ? "\n\n" : "") + `[STDERR]\n${result.stderr}`;
  }
  if (result.exitCode !== 0) {
    output += (output ? "\n\n" : "") + `[Exit code: ${result.exitCode}]`;
  }
  return output || "(no output)";
}

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`Basphere SSH MCP Server running (${SSH_USER}@${SSH_HOST})`);
}

main().catch((error) => {
  console.error("Server error:", error);
  process.exit(1);
});
