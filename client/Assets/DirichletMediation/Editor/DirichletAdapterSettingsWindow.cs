using UnityEditor;
using UnityEngine;

namespace Dirichlet.Mediation.Editor
{
    /// <summary>
    /// Editor window for configuring optional adapters (CSJ, GDT).
    /// DirichletAdSDK is always included as the core SDK.
    /// Supports both Android and iOS platforms.
    /// </summary>
    public class DirichletAdapterSettingsWindow : EditorWindow
    {
        // Android Adapter Preferences
        private const string PrefKeyAndroidEnableCsj = "Dirichlet.Android.EnableCSJ";
        private const string PrefKeyAndroidEnableGdt = "Dirichlet.Android.EnableGDT";
        
        // iOS Adapter Preferences (DirichletAdSDK is always enabled)
        private const string PrefKeyIOSEnableCsj = "Dirichlet.iOS.EnableCSJ";
        private const string PrefKeyIOSEnableGdt = "Dirichlet.iOS.EnableGDT";
        
        private bool androidFoldout = true;
        private bool iosFoldout = true;

        [MenuItem("Dirichlet/Adapter Settings", priority = 2)]
        public static void ShowWindow()
        {
            var window = GetWindow<DirichletAdapterSettingsWindow>("Dirichlet Adapter Settings");
            window.minSize = new Vector2(450, 400);
            window.Show();
        }

        private void OnGUI()
        {
            EditorGUILayout.Space(10);
            EditorGUILayout.LabelField("Dirichlet Mediation Adapter Configuration", EditorStyles.boldLabel);
            EditorGUILayout.HelpBox(
                "Configure which adapters to include in Android and iOS builds. " +
                "These settings affect Gradle dependencies (Android) and CocoaPods dependencies (iOS) during build.",
                MessageType.Info);

            EditorGUILayout.Space(10);

            // Android Section
            androidFoldout = EditorGUILayout.BeginFoldoutHeaderGroup(androidFoldout, "Android Adapters");
            if (androidFoldout)
            {
                EditorGUI.indentLevel++;
                
                var androidEnableCsj = EditorPrefs.GetBool(PrefKeyAndroidEnableCsj, true);
                var androidEnableGdt = EditorPrefs.GetBool(PrefKeyAndroidEnableGdt, true);

                EditorGUI.BeginChangeCheck();

                androidEnableCsj = EditorGUILayout.Toggle(
                    new GUIContent("Enable CSJ (穿山甲)", "Include CSJ adapter and SDK in Android build"),
                    androidEnableCsj);

                EditorGUILayout.Space(3);

                androidEnableGdt = EditorGUILayout.Toggle(
                    new GUIContent("Enable GDT (广点通)", "Include GDT adapter and SDK in Android build"),
                    androidEnableGdt);

                if (EditorGUI.EndChangeCheck())
                {
                    EditorPrefs.SetBool(PrefKeyAndroidEnableCsj, androidEnableCsj);
                    EditorPrefs.SetBool(PrefKeyAndroidEnableGdt, androidEnableGdt);
                }

                EditorGUILayout.Space(5);
                EditorGUILayout.LabelField("Current Android Settings:", EditorStyles.miniLabel);
                EditorGUILayout.LabelField($"  CSJ: {(androidEnableCsj ? "✓ Enabled" : "✗ Disabled")}", EditorStyles.miniLabel);
                EditorGUILayout.LabelField($"  GDT: {(androidEnableGdt ? "✓ Enabled" : "✗ Disabled")}", EditorStyles.miniLabel);

                EditorGUI.indentLevel--;
            }
            EditorGUILayout.EndFoldoutHeaderGroup();

            EditorGUILayout.Space(10);

            // iOS Section
            iosFoldout = EditorGUILayout.BeginFoldoutHeaderGroup(iosFoldout, "iOS Adapters");
            if (iosFoldout)
            {
                EditorGUI.indentLevel++;
                
                var iosEnableCsj = EditorPrefs.GetBool(PrefKeyIOSEnableCsj, true);
                var iosEnableGdt = EditorPrefs.GetBool(PrefKeyIOSEnableGdt, true);

                EditorGUI.BeginChangeCheck();

                iosEnableCsj = EditorGUILayout.Toggle(
                    new GUIContent("Enable CSJ (穿山甲)", "Include CSJ adapter in iOS build via CocoaPods"),
                    iosEnableCsj);

                EditorGUILayout.Space(3);

                iosEnableGdt = EditorGUILayout.Toggle(
                    new GUIContent("Enable GDT (广点通)", "Include GDT adapter in iOS build via CocoaPods"),
                    iosEnableGdt);

                if (EditorGUI.EndChangeCheck())
                {
                    EditorPrefs.SetBool(PrefKeyIOSEnableCsj, iosEnableCsj);
                    EditorPrefs.SetBool(PrefKeyIOSEnableGdt, iosEnableGdt);
                }

                EditorGUILayout.Space(3);

                // DirichletAdSDK is always enabled (required core SDK)
                EditorGUI.BeginDisabledGroup(true);
                EditorGUILayout.Toggle(
                    new GUIContent("Enable DirichletAdSDK", "Core SDK, always included in iOS build via CocoaPods"),
                    true);
                EditorGUI.EndDisabledGroup();

                EditorGUILayout.Space(5);
                EditorGUILayout.LabelField("Current iOS Settings:", EditorStyles.miniLabel);
                EditorGUILayout.LabelField($"  CSJ: {(iosEnableCsj ? "✓ Enabled" : "✗ Disabled")}", EditorStyles.miniLabel);
                EditorGUILayout.LabelField($"  GDT: {(iosEnableGdt ? "✓ Enabled" : "✗ Disabled")}", EditorStyles.miniLabel);
                EditorGUILayout.LabelField("  DirichletAdSDK: ✓ Enabled (Required)", EditorStyles.miniLabel);

                EditorGUI.indentLevel--;
            }
            EditorGUILayout.EndFoldoutHeaderGroup();

            EditorGUILayout.Space(10);
            EditorGUILayout.HelpBox(
                "Note: Changes take effect on the next platform build.\n\n" +
                "• Android: Gradle dependencies will be modified during export.\n" +
                "• iOS: Podfile will be generated dynamically and pod install will run automatically.",
                MessageType.Warning);
        }
    }
}

