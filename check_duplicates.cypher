// Find Field nodes with multiple OF_TYPE relationships
MATCH (f:Field)-[r:OF_TYPE]->(target)
WITH f, count(r) as ofTypeCount, collect(target.type) as targetTypes, collect(labels(target)) as targetLabels
WHERE ofTypeCount > 1
RETURN
  f.name as fieldName,
  f.package as package,
  ofTypeCount,
  targetTypes,
  targetLabels
ORDER BY ofTypeCount DESC
LIMIT 20;
