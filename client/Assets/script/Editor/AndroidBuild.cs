using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEngine;

/// <summary>
/// Android 客户端（APK）打包脚本。
///
/// 提供两个命令行入口（各自输出前会清理目标目录的旧产物）：
/// - AndroidBuild.BuildAndroid   : 非 CLIENT_WS（FxNet 网络库），输出 build_android/tank.apk
/// - AndroidBuild.BuildAndroidWS : CLIENT_WS（WebSocketSharp 网络库），输出 build_android/tank.apk
///
/// 关键点：
/// 1. 目标平台 BuildTarget.Android（普通移动客户端，非专用服务器）。
/// 2. 主场景为 login.unity（构建索引 0），并包含它会跳转加载的 match.unity、tank.unity。
/// 3. 不使用 UNITY_SERVER / AI_RUNNING 宏（那是服务器/AI 机器人专用）。
/// 4. 各入口确定性地开启/关闭 CLIENT_WS（临时修改 Scripting Define Symbols），构建结束后还原，避免污染工程设置。
/// </summary>
public static class AndroidBuild
{
    // 参与打包的场景：login 为主场景（索引 0），会加载 match、tank
    private static readonly string[] k_Scenes =
    {
        "Assets/scene/login.unity",
        "Assets/scene/match.unity",
        "Assets/scene/tank.unity",
    };

    // 输出 APK 文件名（各网络模式共用，构建前清理）
    private const string k_ApkName = "tank.apk";

    // 默认输出目录（项目根下）
    private const string k_OutputSubDir = "build_android";

    // 命令行覆盖输出路径的参数名
    private const string k_OutputFlag = "-androidBuildOutput";

    // CLIENT_WS 宏（WebSocketSharp 网络库）
    private const string k_ClientWsDefine = "CLIENT_WS";

    /// <summary>
    /// 由 build_android/build-android.bat 调用：Android 客户端，非 CLIENT_WS（FxNet 网络库）。
    /// </summary>
    public static void BuildAndroid()
    {
        BuildInternal(enableClientWs: false);
    }

    /// <summary>
    /// 由 build_android/build-android-ws.bat 调用：Android 客户端，CLIENT_WS（WebSocketSharp 网络库）。
    /// </summary>
    public static void BuildAndroidWS()
    {
        BuildInternal(enableClientWs: true);
    }

    /// <summary>
    /// 共用的 Android 构建实现。
    /// </summary>
    /// <param name="enableClientWs">true 开启 CLIENT_WS（WebSocketSharp），false 关闭（FxNet）。</param>
    private static void BuildInternal(bool enableClientWs)
    {
        // 默认输出：<项目根>/build_android/tank.apk
        var outputPath = Path.GetFullPath(
            Path.Combine(Application.dataPath, "..", k_OutputSubDir, k_ApkName));

        // 允许通过命令行覆盖输出路径
        var args = Environment.GetCommandLineArgs();
        for (var i = 0; i < args.Length - 1; i++)
        {
            if (args[i] == k_OutputFlag)
            {
                outputPath = args[i + 1];
                Debug.Log($"[AndroidBuild] Override output path -> {outputPath}");
            }
        }

        var outputDir = Path.GetDirectoryName(outputPath);
        if (!string.IsNullOrEmpty(outputDir) && !Directory.Exists(outputDir))
        {
            Directory.CreateDirectory(outputDir);
        }

        // 打包前清理上一次的构建产物，避免不同版本残留文件混在一起
        CleanPreviousBuild(outputDir, Path.GetFileName(outputPath));

        // Android 的 Scripting Define Symbols 存储在 NamedBuildTarget.Android 下
        var namedTarget = NamedBuildTarget.Android;
        var originalDefines = PlayerSettings.GetScriptingDefineSymbols(namedTarget);

        // 确定性地开启/关闭 CLIENT_WS，使两个构建入口互相独立，不受工程既有设置影响
        var defineList = originalDefines
            .Split(';')
            .Where(s => !string.IsNullOrWhiteSpace(s))
            .ToList();
        var definesChanged = SetDefine(defineList, k_ClientWsDefine, enableClientWs);

        BuildReport report = null;
        try
        {
            if (definesChanged)
            {
                var newDefines = string.Join(";", defineList);
                PlayerSettings.SetScriptingDefineSymbols(namedTarget, newDefines);
                Debug.Log($"[AndroidBuild] Scripting defines for Android -> {newDefines}");
            }
            else
            {
                Debug.Log($"[AndroidBuild] Scripting defines for Android (unchanged) -> {originalDefines}");
            }

            var options = new BuildPlayerOptions
            {
                scenes = k_Scenes,
                locationPathName = outputPath,
                target = BuildTarget.Android,
                targetGroup = BuildTargetGroup.Android,
                options = BuildOptions.None,
            };

            Debug.Log($"[AndroidBuild] Building Android client (CLIENT_WS={enableClientWs}) -> {outputPath}");
            report = BuildPipeline.BuildPlayer(options);
        }
        finally
        {
            // 还原 Scripting Define Symbols，避免污染工程设置
            if (definesChanged)
            {
                PlayerSettings.SetScriptingDefineSymbols(namedTarget, originalDefines);
                Debug.Log($"[AndroidBuild] Scripting defines for Android restored -> {originalDefines}");
            }
        }

        var summary = report.summary;
        if (summary.result == BuildResult.Succeeded)
        {
            Debug.Log($"[AndroidBuild] Build succeeded: {summary.totalSize} bytes, output: {outputPath}");
            EditorApplication.Exit(0);
        }
        else
        {
            var error = $"[AndroidBuild] Build failed: {summary.result}, errors: {summary.totalErrors}";
            Debug.LogError(error);
            Console.Error.WriteLine(error);
            EditorApplication.Exit(1);
        }
    }

    /// <summary>
    /// 清理输出目录中上一次的构建产物（APK 及其符号包），保证目录中只保留最新一次构建的产物。
    /// </summary>
    private static void CleanPreviousBuild(string outputDir, string apkName)
    {
        if (string.IsNullOrEmpty(outputDir) || !Directory.Exists(outputDir))
        {
            return;
        }

        // 当前版本的 APK
        DeleteFileIfExists(Path.Combine(outputDir, apkName));

        // Unity 生成的符号包 / 调试目录
        var apkNoExt = Path.GetFileNameWithoutExtension(apkName);
        foreach (var file in Directory.GetFiles(outputDir, apkNoExt + "*.symbols.zip"))
        {
            DeleteFileIfExists(file);
        }
        foreach (var dir in Directory.GetDirectories(outputDir, "*_BurstDebugInformation_DoNotShip"))
        {
            DeleteDirIfExists(dir);
        }

        Debug.Log($"[AndroidBuild] Cleaned previous build artifacts in {outputDir}");
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
