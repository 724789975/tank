using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using UnityEditor;
using UnityEditor.Android;

namespace Dirichlet.Mediation.Editor
{
    /// <summary>
    /// Post-processes the Gradle files to conditionally include CSJ and GDT adapters
    /// based on EditorPrefs configuration.
    /// </summary>
    public class DirichletGradlePostProcessor : IPostGenerateGradleAndroidProject
    {
        public int callbackOrder => 1;

        public void OnPostGenerateGradleAndroidProject(string path)
        {
            var enableCsj = EditorPrefs.GetBool("Dirichlet.Android.EnableCSJ", true);
            var enableGdt = EditorPrefs.GetBool("Dirichlet.Android.EnableGDT", true);

            ProcessMainTemplateGradle(path, enableCsj, enableGdt);
            ProcessSettingsTemplateGradle(path, enableCsj, enableGdt);
        }

        private void ProcessMainTemplateGradle(string projectPath, bool enableCsj, bool enableGdt)
        {
            // Unity generates gradle files in the project path
            // The path parameter is the root of the generated Gradle project
            var gradlePath = Path.Combine(projectPath, "unityLibrary", "build.gradle");
            
            if (!File.Exists(gradlePath))
            {
                // Fallback: search for build.gradle files
                var gradleFiles = Directory.GetFiles(projectPath, "build.gradle", SearchOption.AllDirectories);
                if (gradleFiles.Length > 0)
                {
                    // Prefer unityLibrary/build.gradle if it exists
                    gradlePath = gradleFiles.FirstOrDefault(f => f.Contains("unityLibrary")) ?? gradleFiles[0];
                }
            }

            if (!File.Exists(gradlePath))
            {
                UnityEngine.Debug.LogWarning($"[Dirichlet] Could not find build.gradle to process in {projectPath}");
                return;
            }

            UnityEngine.Debug.Log($"[Dirichlet] Processing Gradle file: {gradlePath}");

            var content = File.ReadAllText(gradlePath);
            var modified = false;

            // Remove CSJ dependencies if disabled
            if (!enableCsj)
            {
                // Remove CSJ Adapter
                content = Regex.Replace(content,
                    @"^\s*implementation\(name:\s*'DirichletAD_CSJ_Adapter[^']*',\s*ext:\s*'aar'\)\s*$",
                    "",
                    RegexOptions.Multiline);

                // Remove CSJ SDK
                content = Regex.Replace(content,
                    @"^\s*implementation\(name:\s*'open_ad_sdk[^']*',\s*ext:\s*'aar'\)\s*$",
                    "",
                    RegexOptions.Multiline);

                // Remove CSJ Maven repositories
                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://artifact\.bytedance\.com/repository/pangle'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://dlinfra-cdn-tos\.byteimg\.com/union-sdk/tt-adnet/maven/google'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                modified = true;
                UnityEngine.Debug.Log("[Dirichlet] CSJ adapter and SDK removed from Gradle dependencies");
            }

            // Remove GDT dependencies if disabled
            if (!enableGdt)
            {
                // Remove GDT Adapter
                content = Regex.Replace(content,
                    @"^\s*implementation\(name:\s*'DirichletAD_GDT_Adapter[^']*',\s*ext:\s*'aar'\)\s*$",
                    "",
                    RegexOptions.Multiline);

                // Remove GDT SDK
                content = Regex.Replace(content,
                    @"^\s*implementation\(name:\s*'GDTSDK[^']*',\s*ext:\s*'aar'\)\s*$",
                    "",
                    RegexOptions.Multiline);

                // Remove GDT Maven repository
                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://mirrors\.tencent\.com/nexus/repository/maven-public/'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                modified = true;
                UnityEngine.Debug.Log("[Dirichlet] GDT adapter and SDK removed from Gradle dependencies");
            }

            if (modified)
            {
                // Clean up multiple empty lines
                content = Regex.Replace(content, @"\n\s*\n\s*\n", "\n\n", RegexOptions.Multiline);
                File.WriteAllText(gradlePath, content);
            }
        }

        private void ProcessSettingsTemplateGradle(string projectPath, bool enableCsj, bool enableGdt)
        {
            var settingsPath = Path.Combine(projectPath, "settings.gradle");
            if (!File.Exists(settingsPath))
            {
                return;
            }

            var content = File.ReadAllText(settingsPath);
            var modified = false;

            // Remove CSJ Maven repositories if disabled
            if (!enableCsj)
            {
                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://artifact\.bytedance\.com/repository/pangle'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://dlinfra-cdn-tos\.byteimg\.com/union-sdk/tt-adnet/maven/google'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                modified = true;
            }

            // Remove GDT Maven repository if disabled
            if (!enableGdt)
            {
                content = Regex.Replace(content,
                    @"^\s*maven\s*\{\s*url\s*'https://mirrors\.tencent\.com/nexus/repository/maven-public/'\s*\}\s*$",
                    "",
                    RegexOptions.Multiline);

                modified = true;
            }

            if (modified)
            {
                // Clean up multiple empty lines
                content = Regex.Replace(content, @"\n\s*\n\s*\n", "\n\n", RegexOptions.Multiline);
                File.WriteAllText(settingsPath, content);
            }
        }
    }
}

