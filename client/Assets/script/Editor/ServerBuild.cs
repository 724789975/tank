using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEngine;

/// <summary>
/// Linux 专用服务器（Dedicated Server）打包脚本。
///
/// 提供四个命令行入口（各自输出前会清理目标目录的旧产物）：
/// - ServerBuild.BuildLinuxServer   : 游戏服务器，非 CLIENT_WS（FxNet），场景 server.unity，输出 build_ls/tank.x86_64
/// - ServerBuild.BuildLinuxServerWS : 游戏服务器，CLIENT_WS（WebSocketSharp），场景 server.unity，输出 build_ls/tank.x86_64
/// - ServerBuild.BuildLinuxAI       : AI 机器人，非 CLIENT_WS（FxNet），场景 ai.unity，输出 build_lai/tank.x86_64
/// - ServerBuild.BuildLinuxAIWS     : AI 机器人，CLIENT_WS（WebSocketSharp），场景 ai.unity，输出 build_lai/tank.x86_64
///
/// 关键点：
/// 1. 目标平台 StandaloneLinux64（x86_64）。
/// 2. 使用 StandaloneBuildSubtarget.Server 构建专用服务器，该子目标会自动定义 UNITY_SERVER 宏。
/// 3. AI 机器人额外开启 AI_RUNNING 宏（与 UNITY_SERVER 同时生效）。
/// 4. 各入口确定性地开启/关闭 CLIENT_WS、AI_RUNNING（临时修改 Scripting Define Symbols），构建结束后还原，避免污染工程设置。
/// </summary>
public static class ServerBuild
{
    // 游戏服务器场景
    private const string k_ServerScene = "Assets/scene/server.unity";

    // AI 机器人场景
    private const string k_AiScene = "Assets/scene/ai.unity";

    // 可执行文件名，需与 Dockerfile 中 chmod +x ./tank.x86_64 保持一致（各版本共用，构建前清理）
    private const string k_ExecutableName = "tank.x86_64";

    // 命令行覆盖输出路径的参数名
    private const string k_OutputFlag = "-serverBuildOutput";

    // CLIENT_WS 宏（WebSocketSharp 网络库）
    private const string k_ClientWsDefine = "CLIENT_WS";

    // AI_RUNNING 宏（AI 机器人）
    private const string k_AiRunningDefine = "AI_RUNNING";

    /// <summary>
    /// 由 build_ls/build-server.bat 调用：游戏服务器，非 CLIENT_WS（FxNet 网络库）。
    /// </summary>
    public static void BuildLinuxServer()
    {
        BuildInternal(k_ServerScene, "build_ls", k_ExecutableName, enableClientWs: false, enableAiRunning: false);
    }

    /// <summary>
    /// 由 build_ls/build-server-ws.bat 调用：游戏服务器，CLIENT_WS（WebSocketSharp 网络库）。
    /// </summary>
    public static void BuildLinuxServerWS()
    {
        BuildInternal(k_ServerScene, "build_ls", k_ExecutableName, enableClientWs: true, enableAiRunning: false);
    }

    /// <summary>
    /// 由 build_lai/build-ai.bat 调用：AI 机器人，非 CLIENT_WS（FxNet 网络库）。
    /// </summary>
    public static void BuildLinuxAI()
    {
        BuildInternal(k_AiScene, "build_lai", k_ExecutableName, enableClientWs: false, enableAiRunning: true);
    }

    /// <summary>
    /// 由 build_lai/build-ai-ws.bat 调用：AI 机器人，CLIENT_WS（WebSocketSharp 网络库）。
    /// </summary>
    public static void BuildLinuxAIWS()
    {
        BuildInternal(k_AiScene, "build_lai", k_ExecutableName, enableClientWs: true, enableAiRunning: true);
    }

    /// <summary>
    /// 共用的构建实现。
    /// </summary>
    /// <param name="scene">主场景路径。</param>
    /// <param name="outputSubDir">默认输出目录（项目根下的子目录，如 build_ls / build_lai）。</param>
    /// <param name="executableName">默认输出的可执行文件名。</param>
    /// <param name="enableClientWs">true 开启 CLIENT_WS（WebSocketSharp），false 关闭（FxNet）。</param>
    /// <param name="enableAiRunning">true 开启 AI_RUNNING（AI 机器人），false 关闭（游戏服务器）。</param>
    private static void BuildInternal(string scene, string outputSubDir, string executableName, bool enableClientWs, bool enableAiRunning)
    {
        // 默认输出：<项目根>/<outputSubDir>/<executableName>
        var outputPath = Path.GetFullPath(
            Path.Combine(Application.dataPath, "..", outputSubDir, executableName));

        // 允许通过命令行覆盖输出路径
        var args = Environment.GetCommandLineArgs();
        for (var i = 0; i < args.Length - 1; i++)
        {
            if (args[i] == k_OutputFlag)
            {
                outputPath = args[i + 1];
                Debug.Log($"[ServerBuild] Override output path -> {outputPath}");
            }
        }

        var outputDir = Path.GetDirectoryName(outputPath);
        if (!string.IsNullOrEmpty(outputDir) && !Directory.Exists(outputDir))
        {
            Directory.CreateDirectory(outputDir);
        }

        // 打包前清理上一次的构建产物，避免不同版本残留文件混在一起
        CleanPreviousBuild(outputDir, Path.GetFileName(outputPath));

        // 以专用服务器（Server）子目标构建，Unity 会自动定义 UNITY_SERVER 宏
        EditorUserBuildSettings.standaloneBuildSubtarget = StandaloneBuildSubtarget.Server;

        // 专用服务器的 Scripting Define Symbols 存储在 NamedBuildTarget.Server 下
        var namedTarget = NamedBuildTarget.Server;
        var originalDefines = PlayerSettings.GetScriptingDefineSymbols(namedTarget);

        // 确定性地开启/关闭 CLIENT_WS、AI_RUNNING，使各构建入口互相独立，不受工程既有设置影响
        var defineList = originalDefines
            .Split(';')
            .Where(s => !string.IsNullOrWhiteSpace(s))
            .ToList();
        var definesChanged = false;
        definesChanged |= SetDefine(defineList, k_ClientWsDefine, enableClientWs);
        definesChanged |= SetDefine(defineList, k_AiRunningDefine, enableAiRunning);

        BuildReport report = null;
        try
        {
            if (definesChanged)
            {
                var newDefines = string.Join(";", defineList);
                PlayerSettings.SetScriptingDefineSymbols(namedTarget, newDefines);
                Debug.Log($"[ServerBuild] Scripting defines for Server -> {newDefines}");
            }
            else
            {
                Debug.Log($"[ServerBuild] Scripting defines for Server (unchanged) -> {originalDefines}");
            }

            var options = new BuildPlayerOptions
            {
                scenes = new[] { scene },
                locationPathName = outputPath,
                target = BuildTarget.StandaloneLinux64,
                targetGroup = BuildTargetGroup.Standalone,
                subtarget = (int)StandaloneBuildSubtarget.Server,
                options = BuildOptions.None,
            };

            Debug.Log($"[ServerBuild] Building Linux dedicated server (CLIENT_WS={enableClientWs}, AI_RUNNING={enableAiRunning}) -> {outputPath}");
            report = BuildPipeline.BuildPlayer(options);
        }
        finally
        {
            // 还原 Scripting Define Symbols，避免污染工程设置
            if (definesChanged)
            {
                PlayerSettings.SetScriptingDefineSymbols(namedTarget, originalDefines);
                Debug.Log($"[ServerBuild] Scripting defines for Server restored -> {originalDefines}");
            }
        }

        var summary = report.summary;
        if (summary.result == BuildResult.Succeeded)
        {
            Debug.Log($"[ServerBuild] Build succeeded: {summary.totalSize} bytes, output: {outputPath}");
            EditorApplication.Exit(0);
        }
        else
        {
            var error = $"[ServerBuild] Build failed: {summary.result}, errors: {summary.totalErrors}";
            Debug.LogError(error);
            Console.Error.WriteLine(error);
            EditorApplication.Exit(1);
        }
    }

    /// <summary>
    /// 清理输出目录中上一次的构建产物（可执行文件、*_Data、UnityPlayer.so、Burst 调试目录，
    /// 以及历史命名的 tank_ws 产物），保证目录中只保留最新一次构建的运行文件。
    /// </summary>
    private static void CleanPreviousBuild(string outputDir, string executableName)
    {
        if (string.IsNullOrEmpty(outputDir) || !Directory.Exists(outputDir))
        {
            return;
        }

        // 当前版本的可执行文件及其 *_Data 目录
        var exeNoExt = Path.GetFileNameWithoutExtension(executableName);
        DeleteFileIfExists(Path.Combine(outputDir, executableName));
        DeleteDirIfExists(Path.Combine(outputDir, exeNoExt + "_Data"));

        // 共享运行时
        DeleteFileIfExists(Path.Combine(outputDir, "UnityPlayer.so"));

        // Burst 调试目录（*_BurstDebugInformation_DoNotShip）
        foreach (var dir in Directory.GetDirectories(outputDir, "*_BurstDebugInformation_DoNotShip"))
        {
            DeleteDirIfExists(dir);
        }

        // 清理历史版本可能残留的 WS 专用产物（现已统一为 tank.x86_64）
        DeleteFileIfExists(Path.Combine(outputDir, "tank_ws.x86_64"));
        DeleteDirIfExists(Path.Combine(outputDir, "tank_ws_Data"));

        Debug.Log($"[ServerBuild] Cleaned previous build artifacts in {outputDir}");
    }

    private static void DeleteFileIfExists(string path)
    {
        if (File.Exists(path))
        {
            File.Delete(path);
        }
    }

    private static void DeleteDirIfExists(string path)
    {
        if (Directory.Exists(path))
        {
            Directory.Delete(path, recursive: true);
        }
    }

    /// <summary>
    /// 确定性地设置某个宏的开关状态；返回是否发生了变化（用于构建后还原）。
    /// </summary>
    private static bool SetDefine(List<string> defineList, string define, bool enabled)
    {
        if (enabled)
        {
            if (defineList.Contains(define))
            {
                return false;
            }
            defineList.Add(define);
            return true;
        }

        return defineList.Remove(define);
    }
}
