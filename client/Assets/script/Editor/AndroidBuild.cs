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
/// - AndroidBuild.BuildAndroid   : 非 CLIENT_WS（FxNet 网络库），输出 build_android/pongpongpong.apk
/// - AndroidBuild.BuildAndroidWS : CLIENT_WS（WebSocketSharp 网络库），输出 build_android/pongpongpong.apk
///
/// 关键点：
/// 1. 目标平台 BuildTarget.Android（普通移动客户端，非专用服务器）。
/// 2. 主场景为 login.unity（构建索引 0），并包含它会跳转加载的 match.unity、tank.unity。
/// 3. 不使用 UNITY_SERVER / AI_RUNNING 宏（那是服务器/AI 机器人专用）。
/// 4. 各入口确定性地开启/关闭 CLIENT_WS（临时修改 Scripting Define Symbols），构建结束后还原，避免污染工程设置。
/// 5. 包名与签名完全跟随用户在 Player Settings 中的配置，脚本不做任何覆盖；
///    若启用了自定义 keystore，batchmode 下密码需通过环境变量 ANDROID_KEYSTORE_PASS /
///    ANDROID_KEYALIAS_PASS 提供（Unity 不持久化密码），缺失时直接报错而不是静默回退到 debug 签名。
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

    // 输出 APK 文件名（各网络模式共用，构建前清理）；
    // 与 Player Settings 的 productName / applicationIdentifier（com.denglixiaoliu.pongpongpong）保持一致
    private const string k_ApkName = "pongpongpong.apk";

    // 默认输出目录（项目根下）
    private const string k_OutputSubDir = "build_android";

    // 命令行覆盖输出路径的参数名
    private const string k_OutputFlag = "-androidBuildOutput";

    // CLIENT_WS 宏（WebSocketSharp 网络库）
    private const string k_ClientWsDefine = "CLIENT_WS";

    // 自定义 keystore 密码的环境变量名（Unity 不会把密码持久化到 ProjectSettings，batchmode 下需外部传入）
    private const string k_KeystorePassEnv = "ANDROID_KEYSTORE_PASS";
    private const string k_KeyaliasPassEnv = "ANDROID_KEYALIAS_PASS";

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

        // 包名与签名跟随用户的 Player Settings；启用自定义 keystore 时补齐密码，保证签名与用户设置一致
        if (!ApplyUserPackageAndSigningSettings())
        {
            EditorApplication.Exit(1);
            return;
        }

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
    /// 校验并应用用户在 Player Settings 中配置的包名与签名，保证命令行构建产物与编辑器内构建一致：
    /// - 包名：直接使用 Player Settings 中的 applicationIdentifier，不做覆盖，仅输出日志便于核对；
    /// - 签名：未启用自定义 keystore 时与编辑器一致使用 debug keystore；
    ///   启用时因 Unity 不持久化密码，需从环境变量读取并填入，缺失则报错终止（避免静默回退成 debug 签名）。
    /// </summary>
    /// <returns>true 表示签名配置就绪，可以继续构建；false 表示配置缺失，构建应终止。</returns>
    private static bool ApplyUserPackageAndSigningSettings()
    {
        var packageName = PlayerSettings.GetApplicationIdentifier(NamedBuildTarget.Android);
        Debug.Log($"[AndroidBuild] Package name (from Player Settings): {packageName}");

        if (!PlayerSettings.Android.useCustomKeystore)
        {
            // 与编辑器内直接 Build 一致：使用 Unity 默认 debug keystore 签名
            Debug.Log("[AndroidBuild] Custom keystore disabled in Player Settings, signing with default debug keystore");
            return true;
        }

        // keystore 路径与 alias 已随 ProjectSettings 保存，这里只需补齐密码
        var keystoreName = PlayerSettings.Android.keystoreName;
        var keyaliasName = PlayerSettings.Android.keyaliasName;
        var keystorePass = Environment.GetEnvironmentVariable(k_KeystorePassEnv);
        var keyaliasPass = Environment.GetEnvironmentVariable(k_KeyaliasPassEnv);

        if (string.IsNullOrEmpty(keystorePass))
        {
            var error = $"[AndroidBuild] Player Settings 启用了自定义 keystore（{keystoreName}, alias: {keyaliasName}），" +
                        $"但未提供密码。请先设置环境变量 {k_KeystorePassEnv}（及可选的 {k_KeyaliasPassEnv}）后再运行打包脚本，" +
                        "否则产物签名将与用户设置不符。";
            Debug.LogError(error);
            Console.Error.WriteLine(error);
            return false;
        }

        PlayerSettings.Android.keystorePass = keystorePass;
        // alias 密码未单独提供时，按常见做法与 keystore 密码相同
        PlayerSettings.Android.keyaliasPass = string.IsNullOrEmpty(keyaliasPass) ? keystorePass : keyaliasPass;
        Debug.Log($"[AndroidBuild] Signing with custom keystore from Player Settings: {keystoreName} (alias: {keyaliasName})");
        return true;
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
