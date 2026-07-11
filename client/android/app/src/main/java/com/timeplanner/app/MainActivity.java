package com.timeplanner.app;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Context;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.graphics.Typeface;
import android.os.Bundle;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.InputMethodManager;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import android.widget.Toast;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Calendar;
import java.util.List;
import java.util.Locale;

public class MainActivity extends Activity {
    private static final String PREFS = "time_planner";
    private static final String TOKEN = "token";

    private final SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy-MM-dd", Locale.CHINA);
    private SharedPreferences prefs;
    private LinearLayout root;
    private Calendar currentDate;
    private String token;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        token = prefs.getString(TOKEN, "");
        currentDate = Calendar.getInstance();

        if (token == null || token.isEmpty()) {
            showAuthScreen(false);
        } else {
            showMainScreen();
            loadDay();
        }
    }

    private void showAuthScreen(boolean registerMode) {
        root = vertical();
        root.setPadding(dp(24), dp(38), dp(24), dp(24));
        root.setGravity(Gravity.CENTER_VERTICAL);
        root.setBackgroundColor(Color.rgb(247, 248, 251));
        setContentView(root);

        TextView brand = title("时间安排计划", 28);
        TextView sub = muted("登录后同步网页、桌面和手机计划");
        EditText email = input("邮箱");
        EditText password = input("密码");
        password.setInputType(0x00000081);
        Button primary = primaryButton(registerMode ? "创建账号" : "登录");
        Button switchMode = plainButton(registerMode ? "已有账号？登录" : "没有账号？注册");

        root.addView(brand);
        root.addView(sub);
        root.addView(space(18));
        root.addView(email);
        root.addView(space(10));
        root.addView(password);
        root.addView(space(14));
        root.addView(primary);
        root.addView(space(8));
        root.addView(switchMode);

        primary.setOnClickListener(v -> {
            hideKeyboard(password);
            String emailValue = email.getText().toString().trim();
            String passwordValue = password.getText().toString();
            if (emailValue.isEmpty() || passwordValue.length() < 6) {
                toast("请输入邮箱和至少 6 位密码");
                return;
            }
            primary.setEnabled(false);
            primary.setText("处理中...");
            runAsync(() -> {
                try {
                    JSONObject body = new JSONObject();
                    body.put("email", emailValue);
                    body.put("password", passwordValue);
                    JSONObject res = request(registerMode ? "/auth/register" : "/auth/login", body, false);
                    JSONObject data = res.getJSONObject("data");
                    token = data.getString("token");
                    prefs.edit().putString(TOKEN, token).apply();
                    runOnUiThread(() -> {
                        showMainScreen();
                        loadDay();
                    });
                } catch (Exception e) {
                    runOnUiThread(() -> {
                        primary.setEnabled(true);
                        primary.setText(registerMode ? "创建账号" : "登录");
                        toast(e.getMessage());
                    });
                }
            });
        });

        switchMode.setOnClickListener(v -> showAuthScreen(!registerMode));
    }

    private void showMainScreen() {
        root = vertical();
        root.setBackgroundColor(Color.rgb(247, 248, 251));
        setContentView(root);

        LinearLayout header = horizontal();
        header.setPadding(dp(16), dp(16), dp(16), dp(10));
        header.setGravity(Gravity.CENTER_VERTICAL);
        TextView title = title("每日计划", 24);
        Button logout = plainButton("退出");
        header.addView(title, weightParams(1));
        header.addView(logout);
        root.addView(header);

        LinearLayout nav = horizontal();
        nav.setPadding(dp(16), dp(0), dp(16), dp(10));
        Button prev = secondaryButton("上一天");
        Button today = secondaryButton("今天");
        Button next = secondaryButton("下一天");
        nav.addView(prev, weightParams(1));
        nav.addView(today, weightParams(1));
        nav.addView(next, weightParams(1));
        root.addView(nav);

        TextView dateLabel = muted("");
        dateLabel.setId(View.generateViewId());
        dateLabel.setTag("dateLabel");
        dateLabel.setPadding(dp(18), dp(0), dp(18), dp(8));
        root.addView(dateLabel);

        ScrollView scroll = new ScrollView(this);
        LinearLayout list = vertical();
        list.setTag("list");
        list.setPadding(dp(16), dp(0), dp(16), dp(92));
        scroll.addView(list);
        root.addView(scroll, new LinearLayout.LayoutParams(-1, 0, 1));

        Button add = primaryButton("+ 添加计划");
        add.setPadding(dp(18), dp(12), dp(18), dp(12));
        LinearLayout bottom = vertical();
        bottom.setPadding(dp(16), dp(8), dp(16), dp(16));
        bottom.addView(add);
        root.addView(bottom);

        logout.setOnClickListener(v -> {
            prefs.edit().remove(TOKEN).apply();
            token = "";
            showAuthScreen(false);
        });
        prev.setOnClickListener(v -> {
            currentDate.add(Calendar.DAY_OF_MONTH, -1);
            loadDay();
        });
        today.setOnClickListener(v -> {
            currentDate = Calendar.getInstance();
            loadDay();
        });
        next.setOnClickListener(v -> {
            currentDate.add(Calendar.DAY_OF_MONTH, 1);
            loadDay();
        });
        add.setOnClickListener(v -> showAddDialog());
    }

    private void loadDay() {
        TextView dateLabel = findByTag("dateLabel");
        LinearLayout list = findByTag("list");
        String date = dateFormat.format(currentDate.getTime());
        dateLabel.setText(date + "  " + weekday(currentDate));
        list.removeAllViews();
        list.addView(muted("加载中..."));

        runAsync(() -> {
            try {
                JSONObject params = new JSONObject();
                params.put("date", date);
                JSONObject data = api("listDay", params).getJSONObject("data");
                JSONArray items = data.getJSONArray("items");
                List<ScheduleItem> parsed = new ArrayList<>();
                for (int i = 0; i < items.length(); i++) {
                    parsed.add(ScheduleItem.fromJson(items.getJSONObject(i)));
                }
                runOnUiThread(() -> renderItems(parsed));
            } catch (Exception e) {
                runOnUiThread(() -> {
                    if (isAuthExpired(e)) {
                        prefs.edit().remove(TOKEN).apply();
                        showAuthScreen(false);
                    } else {
                        list.removeAllViews();
                        list.addView(empty("加载失败：" + e.getMessage()));
                    }
                });
            }
        });
    }

    private void renderItems(List<ScheduleItem> items) {
        LinearLayout list = findByTag("list");
        list.removeAllViews();
        if (items.isEmpty()) {
            list.addView(empty("今天暂无计划"));
            return;
        }
        for (ScheduleItem item : items) {
            list.addView(itemCard(item));
            list.addView(space(10));
        }
    }

    private View itemCard(ScheduleItem item) {
        LinearLayout card = vertical();
        card.setPadding(dp(14), dp(12), dp(14), dp(12));
        card.setBackgroundColor(Color.WHITE);

        TextView title = title(item.title, 17);
        TextView meta = muted((item.hasTime ? item.startTime + " - " + item.endTime : "全天")
                + "  ·  " + statusLabel(item.status)
                + "  ·  " + priorityLabel(item.priority));
        card.addView(title);
        card.addView(space(4));
        card.addView(meta);
        if (!item.description.isEmpty()) {
            TextView desc = muted(item.description);
            desc.setPadding(0, dp(8), 0, 0);
            card.addView(desc);
        }

        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(-1, -2);
        card.setLayoutParams(lp);
        card.setOnClickListener(v -> showEditDialog(item));
        return card;
    }

    private void showAddDialog() {
        showItemDialog(null);
    }

    private void showEditDialog(ScheduleItem item) {
        showItemDialog(item);
    }

    private void showItemDialog(ScheduleItem existing) {
        boolean edit = existing != null;
        LinearLayout form = vertical();
        form.setPadding(dp(12), dp(8), dp(12), dp(2));
        EditText title = input("标题");
        EditText date = input("日期 yyyy-MM-dd");
        EditText start = input("开始时间 HH:mm");
        EditText end = input("结束时间 HH:mm");
        EditText category = input("分类");
        EditText description = input("描述");

        date.setText(edit ? existing.date : dateFormat.format(currentDate.getTime()));
        start.setText(edit ? existing.startTime : "");
        end.setText(edit ? existing.endTime : "");
        if (edit) {
            title.setText(existing.title);
            category.setText(existing.category);
            description.setText(existing.description);
        }

        form.addView(title);
        form.addView(space(8));
        form.addView(date);
        form.addView(space(8));
        form.addView(start);
        form.addView(space(8));
        form.addView(end);
        form.addView(space(8));
        form.addView(category);
        form.addView(space(8));
        form.addView(description);

        AlertDialog dialog = new AlertDialog.Builder(this)
                .setTitle(edit ? "编辑计划" : "添加计划")
                .setView(form)
                .setNegativeButton("取消", null)
                .setPositiveButton(edit ? "保存" : "添加", null)
                .create();

        dialog.setOnShowListener(d -> dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener(v -> {
            if (title.getText().toString().trim().isEmpty()) {
                toast("请填写标题");
                return;
            }
            saveItem(existing, title, date, start, end, category, description, dialog);
        }));
        dialog.show();
    }

    private void saveItem(ScheduleItem existing, EditText title, EditText date, EditText start, EditText end,
                          EditText category, EditText description, AlertDialog dialog) {
        runAsync(() -> {
            try {
                JSONObject item = new JSONObject();
                item.put("title", title.getText().toString().trim());
                item.put("description", description.getText().toString());
                item.put("date", valueOr(date, dateFormat.format(currentDate.getTime())));
                item.put("startTime", valueOr(start, "00:00"));
                item.put("endTime", valueOr(end, "00:00"));
                item.put("repeat", existing == null ? "none" : existing.repeat);
                item.put("priority", existing == null ? "medium" : existing.priority);
                item.put("status", existing == null ? "pending" : existing.status);
                item.put("category", category.getText().toString());

                JSONObject params = new JSONObject();
                params.put("item", item);
                String action = "addItem";
                if (existing != null) {
                    action = "updateItem";
                    params.put("id", existing.id);
                }
                api(action, params);
                runOnUiThread(() -> {
                    dialog.dismiss();
                    loadDay();
                });
            } catch (Exception e) {
                runOnUiThread(() -> toast(e.getMessage()));
            }
        });
    }

    private JSONObject api(String action, JSONObject params) throws Exception {
        JSONObject body = new JSONObject();
        body.put("action", action);
        body.put("params", params);
        return request("/api", body, true);
    }

    private JSONObject request(String path, JSONObject body, boolean auth) throws Exception {
        URL url = new URL(baseUrl() + trimPath(path));
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        conn.setRequestMethod("POST");
        conn.setConnectTimeout(10000);
        conn.setReadTimeout(15000);
        conn.setRequestProperty("Content-Type", "application/json; charset=utf-8");
        if (auth && token != null && !token.isEmpty()) {
            conn.setRequestProperty("Authorization", "Bearer " + token);
        }
        conn.setDoOutput(true);
        byte[] bytes = body.toString().getBytes(StandardCharsets.UTF_8);
        try (OutputStream os = conn.getOutputStream()) {
            os.write(bytes);
        }

        int code = conn.getResponseCode();
        InputStream stream = code >= 400 ? conn.getErrorStream() : conn.getInputStream();
        String text = readAll(stream);
        JSONObject json = new JSONObject(text);
        if (!json.optBoolean("ok")) {
            throw new RuntimeException(json.optString("error", "请求失败"));
        }
        return json;
    }

    private String baseUrl() {
        String url = BuildConfig.DEFAULT_WEB_URL;
        return url.endsWith("/") ? url.substring(0, url.length() - 1) : url;
    }

    private String trimPath(String path) {
        return path.startsWith("/") ? path : "/" + path;
    }

    private String readAll(InputStream stream) throws Exception {
        if (stream == null) return "";
        BufferedReader reader = new BufferedReader(new InputStreamReader(stream, StandardCharsets.UTF_8));
        StringBuilder sb = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) sb.append(line);
        return sb.toString();
    }

    private boolean isAuthExpired(Exception e) {
        String msg = e.getMessage();
        return msg != null && (msg.contains("登录") || msg.contains("Unauthorized") || msg.contains("401"));
    }

    private String valueOr(EditText input, String fallback) {
        String value = input.getText().toString().trim();
        return value.isEmpty() ? fallback : value;
    }

    private String weekday(Calendar c) {
        return new String[]{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[c.get(Calendar.DAY_OF_WEEK) - 1];
    }

    private String statusLabel(String status) {
        if ("completed".equals(status)) return "已完成";
        if ("in_progress".equals(status)) return "进行中";
        if ("cancelled".equals(status)) return "已取消";
        return "待办";
    }

    private String priorityLabel(String priority) {
        if ("high".equals(priority)) return "高";
        if ("low".equals(priority)) return "低";
        return "中";
    }

    @SuppressWarnings("unchecked")
    private <T extends View> T findByTag(String tag) {
        return (T) root.findViewWithTag(tag);
    }

    private LinearLayout vertical() {
        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        layout.setLayoutParams(new LinearLayout.LayoutParams(-1, -1));
        return layout;
    }

    private LinearLayout horizontal() {
        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.HORIZONTAL);
        return layout;
    }

    private TextView title(String text, int sp) {
        TextView view = new TextView(this);
        view.setText(text);
        view.setTextSize(sp);
        view.setTextColor(Color.rgb(23, 25, 35));
        view.setTypeface(Typeface.DEFAULT, Typeface.BOLD);
        return view;
    }

    private TextView muted(String text) {
        TextView view = new TextView(this);
        view.setText(text);
        view.setTextSize(14);
        view.setTextColor(Color.rgb(104, 115, 134));
        return view;
    }

    private View empty(String text) {
        TextView view = muted(text);
        view.setGravity(Gravity.CENTER);
        view.setPadding(dp(12), dp(34), dp(12), dp(34));
        return view;
    }

    private EditText input(String hint) {
        EditText input = new EditText(this);
        input.setHint(hint);
        input.setTextSize(16);
        input.setSingleLine(false);
        input.setBackgroundColor(Color.WHITE);
        input.setPadding(dp(12), dp(8), dp(12), dp(8));
        input.setLayoutParams(new LinearLayout.LayoutParams(-1, -2));
        return input;
    }

    private Button primaryButton(String text) {
        Button button = new Button(this);
        button.setText(text);
        button.setTextColor(Color.WHITE);
        button.setTextSize(15);
        button.setAllCaps(false);
        button.setBackgroundColor(Color.rgb(79, 125, 240));
        button.setLayoutParams(new LinearLayout.LayoutParams(-1, -2));
        return button;
    }

    private Button secondaryButton(String text) {
        Button button = plainButton(text);
        button.setBackgroundColor(Color.WHITE);
        return button;
    }

    private Button plainButton(String text) {
        Button button = new Button(this);
        button.setText(text);
        button.setTextSize(14);
        button.setAllCaps(false);
        button.setTextColor(Color.rgb(23, 25, 35));
        return button;
    }

    private LinearLayout.LayoutParams weightParams(int weight) {
        return new LinearLayout.LayoutParams(0, -2, weight);
    }

    private View space(int dp) {
        View view = new View(this);
        view.setLayoutParams(new LinearLayout.LayoutParams(1, dp(dp)));
        return view;
    }

    private int dp(int value) {
        return (int) (value * getResources().getDisplayMetrics().density + 0.5f);
    }

    private void toast(String message) {
        Toast.makeText(this, message == null ? "操作失败" : message, Toast.LENGTH_SHORT).show();
    }

    private void hideKeyboard(View view) {
        InputMethodManager imm = (InputMethodManager) getSystemService(Context.INPUT_METHOD_SERVICE);
        if (imm != null) imm.hideSoftInputFromWindow(view.getWindowToken(), 0);
    }

    private void runAsync(Runnable runnable) {
        new Thread(runnable).start();
    }

    private static class ScheduleItem {
        long id;
        String title;
        String description;
        String date;
        String startTime;
        String endTime;
        String repeat;
        String priority;
        String status;
        String category;
        boolean hasTime;

        static ScheduleItem fromJson(JSONObject json) {
            ScheduleItem item = new ScheduleItem();
            item.id = json.optLong("id");
            item.title = json.optString("title");
            item.description = json.optString("description");
            item.date = json.optString("date");
            item.startTime = json.optString("startTime", "00:00");
            item.endTime = json.optString("endTime", "00:00");
            item.repeat = json.optString("repeat", "none");
            item.priority = json.optString("priority", "medium");
            item.status = json.optString("status", "pending");
            item.category = json.optString("category");
            item.hasTime = json.optBoolean("hasTime");
            return item;
        }
    }
}
